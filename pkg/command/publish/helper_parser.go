package publish

import (
	"regexp"
	"strconv"

	"github.com/marco79423/ray/pkg/model"
	"golang.org/x/xerrors"
)

func parseVersion(rawString string) (model.Version, error) {
	r, err := regexp.Compile(`^v?(\d+).?(\d+)?$`)
	if err != nil {
		return model.Version{}, xerrors.Errorf("解析版號失敗: %w", err)
	}

	versions := r.FindStringSubmatch(rawString)
	if len(versions) == 0 {
		return model.Version{}, xerrors.Errorf("解析版號失敗")
	}

	version := model.Version{}

	// 解析主版號
	majorVersion, err := strconv.Atoi(versions[1])
	if err != nil {
		return model.Version{}, xerrors.Errorf("解析版號失敗: %w", err)
	}
	version.Major = majorVersion

	// 解析小版號
	if versions[2] != "" {
		minorVersion, err := strconv.Atoi(versions[2])
		if err != nil {
			return model.Version{}, xerrors.Errorf("解析版號失敗: %w", err)
		}
		version.Minor = minorVersion
	}

	return version, nil
}
