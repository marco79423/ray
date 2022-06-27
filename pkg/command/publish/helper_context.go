package publish

import (
	"context"

	gitSSH "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/marco79423/ray/pkg/util"
	"golang.org/x/xerrors"
)

// 準備所需要的 Context
func prepareContext(options *commandOptions) (context.Context, error) {
	ctx := context.Background()

	// 取得 Repo 路徑
	ctx = context.WithValue(ctx, "repoPath", options.RepoPath)

	// 取得 git auth
	gitAuth, err := util.GetGitAuth(options.PrivateKeyFilePath, options.KeyFilePassword)
	if err != nil {
		return nil, xerrors.Errorf("準備 Context 失敗: %w", err)
	}
	ctx = context.WithValue(ctx, "gitAuth", gitAuth)

	// 開啟 git repo
	gitRepo, err := util.NewGitRepo(
		getCtxGitRepoPath(ctx),
		getCtxGitAuth(ctx),
	)
	ctx = context.WithValue(ctx, "gitRepo", gitRepo)

	return ctx, nil
}

func getCtxGitRepoPath(ctx context.Context) string {
	return ctx.Value("repoPath").(string)
}

func getCtxGitAuth(ctx context.Context) *gitSSH.PublicKeys {
	return ctx.Value("gitAuth").(*gitSSH.PublicKeys)
}

func getCtxGitRepo(ctx context.Context) util.IGitRepository {
	return ctx.Value("gitRepo").(util.IGitRepository)
}
