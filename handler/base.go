package handler

type Handler struct {
	repo Repo
}

func New(repo Repo) *Handler {
	return &Handler{repo: repo}
}
