all: check-soa

check-soa: check-soa.go
	# https://git-scm.com/book/en/v2/Git-Basics-Viewing-the-Commit-History
	go build -ldflags "-X main.Version \"check-soa: $$(git log --date=iso --pretty=format:'Last commit on HEAD %H on %ad' HEAD~..HEAD) Build on $$(date -u '+%Y-%m-%dT%H:%M:%S') UTC\"" check-soa.go
