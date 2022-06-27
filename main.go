package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	gitSSH "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

type ctxOptions struct {
	RepoPath           string
	PrivateKeyFilePath string
	KeyFilePassword    string
}

func main() {
	app := cli.NewApp()
	app.Name = "ray"
	app.Usage = "簡易佈版小幫手"

	app.Commands = []*cli.Command{
		{
			Name:    "publish",
			Usage:   "升版",
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
			Action: func(c *cli.Context) error {
				if c.Args().Len() == 0 {
					fmt.Println("請提供要上版的版號，格式可為 2, v2, 2.0 v2.0, 2.1, v2.1")
					return nil
				}

				ctx, err := prepareContext(&ctxOptions{
					RepoPath:           c.String("path"),
					PrivateKeyFilePath: c.String("keyfile"),
					KeyFilePassword:    c.String("keyfile-password"),
				})
				if err != nil {
					return xerrors.Errorf("程式執行失敗: %w", err)
				}

				rawVersion := c.Args().First()
				if err := publish(ctx, rawVersion); err != nil {
					return xerrors.Errorf("程式執行失敗: %w", err)
				}

				fmt.Println("佈版完畢，讓我們感謝 Ray 領導")
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("%+v", err)
	}
}

// 準備所需要的 Context
func prepareContext(options *ctxOptions) (context.Context, error) {
	ctx := context.Background()

	// 取得 Repo 路徑
	ctx = context.WithValue(ctx, "repoPath", options.RepoPath)

	// 取得 git auth
	gitAuth, err := getGitAuth(options.PrivateKeyFilePath, "")
	if err != nil {
		return nil, xerrors.Errorf("準備 Context 失敗: %w", err)
	}
	ctx = context.WithValue(ctx, "gitAuth", gitAuth)

	return ctx, nil
}

// 取得使用 Git 的權限
func getGitAuth(privateKeyFilePath, password string) (*gitSSH.PublicKeys, error) {
	publicKeys, err := gitSSH.NewPublicKeysFromFile("git", privateKeyFilePath, password)
	if err != nil {
		return nil, xerrors.Errorf("取得 Git Auth 失敗: %w", err)
	}

	publicKeys.HostKeyCallbackHelper = gitSSH.HostKeyCallbackHelper{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return publicKeys, nil
}

// 發佈版本
func publish(ctx context.Context, version string) error {
	magerVersion, minorVersion, err := parseVersion(version)
	if err != nil {
		return nil
	}

	if minorVersion == "0" {
		return publishMagerVersion(ctx, magerVersion)
	} else {
		return publishMinorVersion(ctx, magerVersion, minorVersion)
	}
}

// 發佈主要版本
func publishMagerVersion(ctx context.Context, magerVersion string) error {
	// 切換到 develop
	if err := gitCheckoutTo(ctx, "develop"); err != nil {
		return xerrors.Errorf("發佈主要版本失敗: %w", err)
	}

	// 更新到最新
	if err := gitPull(ctx); err != nil {
		return xerrors.Errorf("發佈主要版本失敗: %w", err)
	}

	// 建立 release 分支
	releaseBranchName := generateReleaseBranchName(magerVersion)
	if err := createBranch(ctx, releaseBranchName); err != nil {
		return xerrors.Errorf("發佈主要版本失敗: %w", err)
	}

	// 建立 tag
	tagName := generateTagName(magerVersion, "0")
	existed, err := gitTagExists(ctx, tagName)
	if err != nil {
		return xerrors.Errorf("發佈主要版本失敗: %w", err)
	}
	if existed {
		return xerrors.Errorf("發佈主要版本失敗: Tag 存在")
	}

	if err := createGitTag(ctx, tagName); err != nil {
		return xerrors.Errorf("發佈主要版本失敗: %w", err)
	}

	// 推送到服務端
	if err := pushBranchAndTag(ctx); err != nil {
		return xerrors.Errorf("發佈主要版本失敗: %w", err)
	}

	return nil
}

// 發佈小版本
func publishMinorVersion(ctx context.Context, magerVersion, minorVersion string) error {
	// 切換到 release
	releaseBranchName := generateReleaseBranchName(magerVersion)
	if err := gitCheckoutTo(ctx, releaseBranchName); err != nil {
		return xerrors.Errorf("發佈小版本失敗: %w", err)
	}

	// 更新到最新
	if err := gitPull(ctx); err != nil {
		return xerrors.Errorf("發佈小版本失敗: %w", err)
	}

	// 建立 tag
	tagName := generateTagName(magerVersion, minorVersion)

	existed, err := gitTagExists(ctx, tagName)
	if err != nil {
		return xerrors.Errorf("發佈小版本失敗: %w", err)
	}
	if existed {
		return xerrors.Errorf("發佈小版本失敗: Tag 存在")
	}

	if err := createGitTag(ctx, tagName); err != nil {
		return xerrors.Errorf("發佈小版本失敗: %w", err)
	}

	// 推送到服務端
	if err := pushBranchAndTag(ctx); err != nil {
		return xerrors.Errorf("發佈小版本失敗: %w", err)
	}
	return nil
}

func generateReleaseBranchName(magerVersion string) string {
	return fmt.Sprintf("release/v%s", magerVersion)
}

func generateTagName(magerVersion, minorVersion string) string {
	return fmt.Sprintf("v%s.%s", magerVersion, minorVersion)
}

func gitCheckoutTo(ctx context.Context, branchName string) error {
	repository, err := openGitRepository(ctx)
	if err != nil {
		return xerrors.Errorf("Git checkout 失敗: %w", err)
	}

	w, err := repository.Worktree()
	if err != nil {
		return xerrors.Errorf("Git checkout 失敗: %w", err)
	}

	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/" + branchName),
		Keep:   true,
	})
	if err != nil {
		return xerrors.Errorf("Git checkout 失敗: %w", err)
	}

	return nil
}

func gitPull(ctx context.Context) error {
	repository, err := openGitRepository(ctx)
	if err != nil {
		return xerrors.Errorf("Git gitPull 失敗: %w", err)
	}

	w, err := repository.Worktree()
	if err != nil {
		return xerrors.Errorf("Git gitPull 失敗: %w", err)
	}

	gitAuth := ctx.Value("gitAuth").(*gitSSH.PublicKeys)
	err = w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth:       gitAuth,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return xerrors.Errorf("Git gitPull 失敗: %w", err)
	}

	return nil
}

func openGitRepository(ctx context.Context) (*git.Repository, error) {
	repoPath := ctx.Value("repoPath").(string)
	repository, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, xerrors.Errorf("取得 Git Repository 失敗: %w", err)
	}
	return repository, err
}

func createGitTag(ctx context.Context, tagName string) error {
	repository, err := openGitRepository(ctx)
	if err != nil {
		return xerrors.Errorf("建立 Git tag 失敗: %w", err)
	}

	headRef, err := repository.Head()
	if err != nil {
		return xerrors.Errorf("建立 Git tag 失敗: %w", err)
	}

	refName := plumbing.ReferenceName("refs/tags/" + tagName)
	ref := plumbing.NewHashReference(refName, headRef.Hash())

	err = repository.Storer.SetReference(ref)
	if err != nil {
		return xerrors.Errorf("建立 Git tag 失敗: %w", err)
	}

	return nil
}

func gitTagExists(ctx context.Context, tagName string) (bool, error) {
	repository, err := openGitRepository(ctx)
	if err != nil {
		return false, xerrors.Errorf("檢查 Git tag 是否存在失敗: %w", err)
	}

	tags, err := repository.Tags()
	if err != nil {
		return false, xerrors.Errorf("檢查 Git tag 是否存在失敗: %w", err)
	}

	existed := false
	for {
		tagRef, err := tags.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return false, xerrors.Errorf("檢查 Git tag 是否存在失敗: %w", err)
		}

		if tagRef.Name() == plumbing.ReferenceName("refs/tags/"+tagName) {
			existed = true
			break
		}
	}

	return existed, nil
}

func pushBranchAndTag(ctx context.Context) error {
	auth := ctx.Value("gitAuth").(*gitSSH.PublicKeys)

	repository, err := openGitRepository(ctx)
	if err != nil {
		return xerrors.Errorf("建立 Git tag 失敗: %w", err)
	}

	err = repository.Push(&git.PushOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
		RefSpecs: []config.RefSpec{
			"+refs/heads/*:refs/remotes/origin/*",
			"refs/tags/*:refs/tags/*",
		},
		Auth: auth,
	})
	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			log.Print("origin remote was up to date, no push done")
			return nil
		}
		log.Printf("push to remote origin error: %s", err)
		return err
	}

	return nil
}

func createBranch(ctx context.Context, branchName string) error {
	repository, err := openGitRepository(ctx)
	if err != nil {
		return xerrors.Errorf("建立 Git branch 失敗: %w", err)
	}

	headRef, err := repository.Head()
	if err != nil {
		return xerrors.Errorf("建立 Git branch 失敗: %w", err)
	}

	refName := plumbing.ReferenceName("refs/heads/" + branchName)
	ref := plumbing.NewHashReference(refName, headRef.Hash())
	err = repository.Storer.SetReference(ref)
	if err != nil {
		return xerrors.Errorf("建立 Git branch 失敗: %w", err)
	}

	return nil
}

func parseVersion(rawString string) (string, string, error) {
	r, err := regexp.Compile(`^v?(\d+).?(\d+)?$`)
	if err != nil {
		return "", "", xerrors.Errorf("解析版號失敗: %w", err)
	}

	versions := r.FindStringSubmatch(rawString)
	if len(versions) == 0 {
		return "", "", xerrors.Errorf("解析版號失敗")
	}

	// 解析主版號
	magerVersion := versions[1]

	// 解析小版號
	minorVersion := versions[2]
	if minorVersion == "" {
		minorVersion = "0"
	}

	return magerVersion, minorVersion, nil
}
