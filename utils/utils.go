package utils

import (
	"math"

	"github.com/xybydy/gdutils/logger"
)

func CheckErr(e error) {
	if e != nil {
		logger.Panic("", e)
	}
}

func Pow(x int, y int) int {
	f := math.Pow(float64(x), float64(y))
	return int(f)
}
