package util

import (
	"errors"
	"io"
	"log"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	gitSSH "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
	"golang.org/x/xerrors"
)

type IGitRepository interface {
	CheckoutTo(branchName string) error
	Pull() error
	CreateTag(tagName string) error
	TagExists(tagName string) (bool, error)
	PushBranchAndTag() error
	CreateBranch(branchName string) error
	GetAllTagNames() ([]string, error)
}

func NewGitRepo(repoPath string, gitAuth *gitSSH.PublicKeys) (IGitRepository, error) {
	repository, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, xerrors.Errorf("取得 Git Repository 失敗: %w", err)
	}

	return &gitRepository{
		repo:    repository,
		gitAuth: gitAuth,
	}, nil
}

func GetGitAuth(privateKeyFilePath, password string) (*gitSSH.PublicKeys, error) {
	publicKeys, err := gitSSH.NewPublicKeysFromFile("git", privateKeyFilePath, password)
	if err != nil {
		return nil, xerrors.Errorf("取得 Git Auth 失敗: %w", err)
	}

	publicKeys.HostKeyCallbackHelper = gitSSH.HostKeyCallbackHelper{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return publicKeys, nil
}

type gitRepository struct {
	repo    *git.Repository
	gitAuth *gitSSH.PublicKeys
}

func (gitRepo *gitRepository) CheckoutTo(branchName string) error {
	w, err := gitRepo.repo.Worktree()
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

func (gitRepo *gitRepository) Pull() error {
	w, err := gitRepo.repo.Worktree()
	if err != nil {
		return xerrors.Errorf("Git gitPull 失敗: %w", err)
	}

	err = w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth:       gitRepo.gitAuth,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return xerrors.Errorf("Git gitPull 失敗: %w", err)
	}

	return nil
}

func (gitRepo *gitRepository) CreateTag(tagName string) error {
	headRef, err := gitRepo.repo.Head()
	if err != nil {
		return xerrors.Errorf("建立 Git tag 失敗: %w", err)
	}

	refName := plumbing.ReferenceName("refs/tags/" + tagName)
	ref := plumbing.NewHashReference(refName, headRef.Hash())

	err = gitRepo.repo.Storer.SetReference(ref)
	if err != nil {
		return xerrors.Errorf("建立 Git tag 失敗: %w", err)
	}

	return nil
}

func (gitRepo *gitRepository) TagExists(tagName string) (bool, error) {
	tags, err := gitRepo.repo.Tags()
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

func (gitRepo *gitRepository) PushBranchAndTag() error {
	err := gitRepo.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
		RefSpecs: []config.RefSpec{
			"+refs/heads/*:refs/remotes/origin/*",
			"refs/tags/*:refs/tags/*",
		},
		Auth: gitRepo.gitAuth,
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

func (gitRepo *gitRepository) CreateBranch(branchName string) error {
	headRef, err := gitRepo.repo.Head()
	if err != nil {
		return xerrors.Errorf("建立 Git branch 失敗: %w", err)
	}

	refName := plumbing.ReferenceName("refs/heads/" + branchName)
	ref := plumbing.NewHashReference(refName, headRef.Hash())
	err = gitRepo.repo.Storer.SetReference(ref)
	if err != nil {
		return xerrors.Errorf("建立 Git branch 失敗: %w", err)
	}

	return nil
}

func (gitRepo *gitRepository) GetAllTagNames() ([]string, error) {
	tags, err := gitRepo.repo.Tags()
	if err != nil {
		return nil, xerrors.Errorf("取得 Git tag 列表失敗: %w", err)
	}

	var tagsNames []string
	for {
		tagRef, err := tags.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, xerrors.Errorf("取得 Git tag 列表失敗: %w", err)
		}
		tagsNames = append(tagsNames, tagRef.Name().Short())
	}

	return tagsNames, nil
}
