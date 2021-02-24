package gd

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github.com/xybydy/gdutils/auth"
	"github.com/xybydy/gdutils/config"
	"github.com/xybydy/gdutils/logger"
	"github.com/xybydy/gdutils/utils"
)

func driveCall(ctx context.Context, fid string) (*drive.Drive, error) {
	logger.Debug("%s - %s", "Drivecall request call args", fid)

	for retry := 0; retry <= config.RetryLimit; retry++ {
		select {
		case <-ctx.Done():
			logger.Debug("", "Cancelled by user")
			return nil, errors.Errorf("Cancelled by user")
		default:
			saFile, err := SaConfigs.UseSa()
			if err != nil {
				logger.Error("", err)
			}
			client := auth.NewServiceAccountClient(ctx, saFile)

			service, err := drive.NewService(ctx, option.WithHTTPClient(client))
			if err != nil {
				logger.Error("", err)
				return nil, err
			}
			logger.Debug("", "New service created")

			f, err := service.Drives.Get(fid).Do()
			if err != nil {
				switch {
				case utils.IsRateLimitError(err):
					SaConfigs.DecSa()
					continue
				case utils.IsBackendError(err):
					SaConfigs.MarkFinished(saFile)
					continue
				default:
					SaConfigs.MarkFinished(saFile)
					return nil, err
				}
			}
			SaConfigs.MarkFinished(saFile)
			return f, err
		}
	}
	return nil, errors.New("no chance to drive call")
}

func fileGetCall(ctx context.Context, fid string, args ListArgs) (*drive.File, error) {
	logger.Debug("%s - %s - %s", "FileGetCall request call args", fid, args)
	for retry := 0; retry <= config.RetryLimit; retry++ {
		select {
		case <-ctx.Done():
			logger.Debug("", "Cancelled by user")
			return nil, errors.Errorf("Cancelled by user")
		default:
			saFile, err := SaConfigs.UseSa()
			if err != nil {
				logger.Error("", err)
			}
			client := auth.NewServiceAccountClient(ctx, saFile)

			service, err := drive.NewService(ctx, option.WithHTTPClient(client))
			if err != nil {
				logger.Error("", err)
				return nil, err
			}
			logger.Debug("", "New service created")

			f, err := service.Files.Get(fid).SupportsAllDrives(true).Fields(args.Fields...).Do()
			if err != nil {
				switch {
				case utils.IsRateLimitError(err):
					SaConfigs.DecSa()
					continue
				case utils.IsBackendError(err):
					SaConfigs.MarkFinished(saFile)
					continue
				default:
					SaConfigs.MarkFinished(saFile)
					return nil, err
				}
			}
			SaConfigs.MarkFinished(saFile)
			return f, err
		}
	}
	return nil, errors.New("no chance to file get call")
}

func fileCreateCall(ctx context.Context, file *drive.File, args ListArgs) (*drive.File, error) {
	logger.Debug("%s - %s - %s", "FileCreateCall request call args", file.Name, args)
	for retry := 0; retry <= config.RetryLimit; retry++ {
		select {
		case <-ctx.Done():
			logger.Debug("", "Cancelled by user")
			return nil, errors.Errorf("Cancelled by user")
		default:
			saFile, err := SaConfigs.UseSa()
			if err != nil {
				logger.Error("", err)
			}

			client := auth.NewServiceAccountClient(ctx, saFile)

			service, err := drive.NewService(ctx, option.WithHTTPClient(client))
			if err != nil {
				logger.Error("", err)
				return nil, err
			}

			f, err := service.Files.Create(file).SupportsAllDrives(args.supportsAllDrives).Do()
			if err != nil {
				switch {
				case utils.IsRateLimitError(err):
					SaConfigs.DecSa()
					continue
				case utils.IsBackendError(err):
					SaConfigs.MarkFinished(saFile)
					continue
				default:
					SaConfigs.MarkFinished(saFile)
					return nil, err
				}
			}
			SaConfigs.MarkFinished(saFile)
			return f, err
		}
	}
	return nil, errors.New("no chance to file create call")
}

func fileListCall(ctx context.Context, args ListArgs) ([]*drive.File, error) {
	var files []*drive.File
	pageSize := int64(config.PageSize)

	if pageSize > 1000 {
		pageSize = 1000
	}

	logger.Debug("%s - %s", "FileListCall request call args", args)
	for retry := 0; retry <= config.RetryLimit; retry++ {
		select {
		case <-ctx.Done():
			logger.Debug("", "Cancelled by user")
			return nil, errors.Errorf("Cancelled by user")
		default:
			saFile, err := SaConfigs.UseSa()
			if err != nil {
				logger.Error("", err)
			}
			client := auth.NewServiceAccountClient(ctx, saFile)

			service, err := drive.NewService(ctx, option.WithHTTPClient(client))
			if err != nil {
				logger.Error("", err)
				return nil, err
			}
			logger.Debug("", "New service created")

			err = service.Files.List().IncludeItemsFromAllDrives(args.includeItemsFromAllDrives).
				SupportsAllDrives(args.supportsAllDrives).Q(args.Query).Fields(args.Fields...).
				OrderBy(args.SortOrder).PageSize(pageSize).Pages(ctx, func(fileList *drive.FileList) error {
				files = append(files, fileList.Files...)
				return nil
			})
			if err != nil {
				switch {
				case utils.IsRateLimitError(err):
					SaConfigs.DecSa()
					continue
				case utils.IsBackendError(err):
					SaConfigs.MarkFinished(saFile)
					continue
				default:
					SaConfigs.MarkFinished(saFile)
					return nil, err
				}
			}
			SaConfigs.MarkFinished(saFile)
			return files, err
		}
	}
	return nil, errors.New("no chance to file list call")
}

func fileCopyCall(ctx context.Context, id, parent string, args ListArgs) (*drive.File, error) {
	logger.Debug("%s - ID: %s - Parent: %s - Args: %s", "fileCopyCall request call args", id, parent, args)
	for retry := 0; retry < config.RetryLimit; retry++ {
		select {
		case <-ctx.Done():
			logger.Debug("", "Request cancelled by user.")
			return nil, errors.Errorf("Cancelled by user")
		default:
			saFile, err := SaConfigs.UseSa()
			if err != nil {
				logger.Error("", err)
			}
			client := auth.NewServiceAccountClient(ctx, saFile)

			service, err := drive.NewService(ctx, option.WithHTTPClient(client))
			if err != nil {
				return nil, err
			}

			f := &drive.File{Parents: []string{parent}}

			file, err := service.Files.Copy(id, f).SupportsAllDrives(args.supportsAllDrives).Do()
			if err != nil {
				switch {
				case utils.IsRateLimitError(err):
					SaConfigs.DecSa()
					continue
				case utils.IsBackendError(err):
					SaConfigs.MarkFinished(saFile)
					continue
				default:
					SaConfigs.MarkFinished(saFile)
					return nil, err
				}
			}
			SaConfigs.MarkFinished(saFile)
			return file, err
		}
	}
	return nil, errors.New("no chance to file copy call")
}
