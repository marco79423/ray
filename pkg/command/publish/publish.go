package publish

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/c-bata/go-prompt"
	"github.com/marco79423/ray/pkg/model"
	"github.com/marco79423/ray/pkg/util"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

type commandOptions struct {
	RepoPath           string
	PrivateKeyFilePath string
	KeyFilePassword    string
}

func Command() *cli.Command {
	return &cli.Command{
		Name:    "publish",
		Usage:   "發布版本",
		Aliases: []string{"worship", "p", "fuck"},
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:    "path",
				Aliases: []string{"p"},
				Usage:   "Repository 路徑",
				Value:   ".",
			},
			&cli.PathFlag{
				Name:    "keyfile",
				Aliases: []string{"f"},
				Usage:   "Private Key 檔案路徑",
				Value:   filepath.Join(os.Getenv("USERPROFILE"), ".ssh", "id_rsa"),
			},
			&cli.PathFlag{
				Name:  "keyfile-password",
				Usage: "Private Key 的密碼",
				Value: "",
			},
		},
		ArgsUsage: "[版號]",
		Action: func(c *cli.Context) error {
			ctx, err := prepareContext(&commandOptions{
				RepoPath:           c.String("path"),
				PrivateKeyFilePath: c.String("keyfile"),
				KeyFilePassword:    c.String("keyfile-password"),
			})
			if err != nil {
				return xerrors.Errorf("程式執行失敗: %w", err)
			}

			rawVersion := c.Args().First()
			for rawVersion == "" {
				fmt.Println("請提供要上版的版號，格式可為 2, v2, 2.0 v2.0, 2.1, v2.1")

				latestVersion, err := getLatestVersion(ctx)
				if err != nil {
					return xerrors.Errorf("程式執行失敗: %w", err)
				}
				rawVersion = prompt.Input("> ", func(d prompt.Document) []prompt.Suggest {
					s := []prompt.Suggest{
						{Text: latestVersion.NextMajor().String(), Description: "下一個主要版本"},
						{Text: latestVersion.NextMinor().String(), Description: "下一個次要版本"},
					}
					return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
				})
			}

			version, err := parseVersion(rawVersion)
			if err != nil {
				return xerrors.Errorf("程式執行失敗: %w", err)
			}

			if err := publish(ctx, version); err != nil {
				return xerrors.Errorf("程式執行失敗: %w", err)
			}

			fmt.Printf("%s 發布版本成功，讓我們感謝 Ray 領導", version.String())
			return nil
		},
	}
}

func getLatestVersion(ctx context.Context) (model.Version, error) {
	// 取得 Git repo
	gitRepo := getCtxGitRepo(ctx)

	allGitTagNames, err := gitRepo.GetAllTagNames()
	if err != nil {
		return model.Version{}, xerrors.Errorf("取得最新的 Git tag 失敗: %w", err)
	}

	// TODO: 濾掉不合法的版號

	if len(allGitTagNames) == 0 {
		return model.Version{}, nil
	}

	sort.Strings(allGitTagNames)

	latestVersion := allGitTagNames[len(allGitTagNames)-1]
	version, err := parseVersion(latestVersion)
	if err != nil {
		return model.Version{}, xerrors.Errorf("取得最新的 Git tag 失敗: %w", err)
	}

	return version, nil
}

// 發布版本
func publish(ctx context.Context, version model.Version) error {
	// 取得 Git repo
	gitRepo := getCtxGitRepo(ctx)
	if version.Minor == 0 {
		return publishMajorVersion(gitRepo, version)
	} else {
		return publishMinorVersion(gitRepo, version)
	}
}

// 發布主要版本
func publishMajorVersion(gitRepo util.IGitRepository, version model.Version) error {
	// 切換到 develop
	if err := gitRepo.CheckoutTo("develop"); err != nil {
		return xerrors.Errorf("發布主要版本失敗: %w", err)
	}

	// 更新到最新
	if err := gitRepo.Pull(); err != nil {
		return xerrors.Errorf("發布主要版本失敗: %w", err)
	}

	// 建立 release 分支
	releaseBranchName := generateReleaseBranchName(version)
	if err := gitRepo.CreateBranch(releaseBranchName); err != nil {
		return xerrors.Errorf("發布主要版本失敗: %w", err)
	}

	// 建立 tag
	existed, err := gitRepo.TagExists(version.String())
	if err != nil {
		return xerrors.Errorf("發布主要版本失敗: %w", err)
	}
	if existed {
		return xerrors.Errorf("發布主要版本失敗: Tag 存在")
	}

	if err := gitRepo.CreateTag(version.String()); err != nil {
		return xerrors.Errorf("發布主要版本失敗: %w", err)
	}

	// 推送到服務端
	if err := gitRepo.PushBranchAndTag(); err != nil {
		return xerrors.Errorf("發布主要版本失敗: %w", err)
	}

	return nil
}

// 發布次要版本
func publishMinorVersion(gitRepo util.IGitRepository, version model.Version) error {
	// 切換到 release
	releaseBranchName := generateReleaseBranchName(version)
	if err := gitRepo.CheckoutTo(releaseBranchName); err != nil {
		return xerrors.Errorf("發布小版本失敗: %w", err)
	}

	// 更新到最新
	if err := gitRepo.Pull(); err != nil {
		return xerrors.Errorf("發布小版本失敗: %w", err)
	}

	// 建立 tag
	existed, err := gitRepo.TagExists(version.String())
	if err != nil {
		return xerrors.Errorf("發布小版本失敗: %w", err)
	}
	if existed {
		return xerrors.Errorf("發布小版本失敗: Tag 存在")
	}

	if err := gitRepo.CreateTag(version.String()); err != nil {
		return xerrors.Errorf("發布小版本失敗: %w", err)
	}

	// 推送到服務端
	if err := gitRepo.PushBranchAndTag(); err != nil {
		return xerrors.Errorf("發布小版本失敗: %w", err)
	}
	return nil
}

func generateReleaseBranchName(version model.Version) string {
	return fmt.Sprintf("release/v%d", version.Major)
}
