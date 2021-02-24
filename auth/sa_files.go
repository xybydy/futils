package auth

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/oauth2/jwt"

	"github.com/xybydy/gdutils/config"
	"github.com/xybydy/gdutils/counter"
	"github.com/xybydy/gdutils/logger"
)

type ServiceAccounter interface {
	InitFiles(string) error
	RefreshActive() error
	UseSa() *jwt.Config
	MarkFinished(config *jwt.Config)
}

type SaFileOrganizer struct {
	availableFiles [][]byte
	// Sa files actively in use
	activeSA    chan *jwt.Config
	activeSANum counter.Counter

	rawFilePath []string
}

func (s *SaFileOrganizer) fetchSaFiles(saLocation string) error {
	rootPath, err := os.Getwd()
	logger.Debug("", "Reading SA Files on ", rootPath)
	if err != nil {
		return err
	}

	saPath := filepath.Join(rootPath, saLocation, "*.json")
	s.rawFilePath, err = filepath.Glob(saPath)
	logger.Debug("%d %s", len(s.rawFilePath), "of sa files found")
	if err != nil {
		return err
	}
	return nil
}

func (s *SaFileOrganizer) InitFiles(saLocation string) error {
	s.activeSA = make(chan *jwt.Config, config.ParallelLimit)

	if err := s.fetchSaFiles(saLocation); err != nil {
		return err
	}

	for i, f := range s.rawFilePath {
		c, err := NewServiceAccountFile(f)
		if err != nil {
			logger.Error("%s", err)
		}
		if i < config.ParallelLimit {
			f, err := NewServiceAccount(c)
			if err != nil {
				logger.Error("%s", err)
			}

			q, err := f.TokenSource(context.TODO()).Token()
			if err != nil {
				logger.Error("%s", err)
			}

			if q.Valid() {
				s.activeSA <- f
				s.activeSANum.Inc()
			}
		} else {
			s.availableFiles = append(s.availableFiles, c)
		}
	}
	logger.Debug("%d of SA files marked as active ", s.activeSANum.Get())
	return nil
}

func (s *SaFileOrganizer) RefreshActive() error {
	var f []byte
	if len(s.availableFiles) == 0 {
		// log.Println("No available SA")
		return errors.New("no available SA")
	}

	for len(s.activeSA) < config.ParallelLimit {
		f, s.availableFiles = s.availableFiles[len(s.availableFiles)-1], s.availableFiles[:len(s.availableFiles)-1]
		c, err := NewServiceAccount(f)
		if err != nil {
			logger.Error("%s", err)
		}

		q, err := c.TokenSource(context.TODO()).Token()
		if err != nil {
			logger.Error("%s", err)
		}

		if q.Valid() {
			logger.Debug("Valid SA, adding to orchestra")
			s.activeSA <- c
			s.activeSANum.Inc()
			break
		}
	}
	return nil
}

// Pendingen bir tane alir aktife koyar ve return
func (s *SaFileOrganizer) UseSa() (*jwt.Config, error) {
	logger.Debug("Working no of SA: %d", s.activeSANum.Get())
	if s.activeSANum.Get() < config.ParallelLimit {
		logger.Debug("", "Adding a new sa to active list")
		if err := s.RefreshActive(); err != nil && s.activeSANum.Get() == 0 {
			return nil, err
		}
	}
	sa := <-s.activeSA
	logger.Debug("Using Sa: %s", sa.Email)
	return sa, nil
}

func (s *SaFileOrganizer) MarkFinished(config *jwt.Config) {
	s.activeSA <- config
	logger.Debug("Sa got back to orchestra: %s ", config.Email)
}

func (s *SaFileOrganizer) DecSa() {
	logger.Debug("A SA went to garbage, remaining no of SA: %d", len(s.activeSA)+len(s.availableFiles))
	s.activeSANum.Dec()
}
