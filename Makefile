all: check-soa

check-soa: check-soa.go
	# https://git-scm.com/book/en/v2/Git-Basics-Viewing-the-Commit-History
	go build -ldflags "-X \"main.lVersion=check-soa: $$(git log --date=iso --pretty=format:'Last commit on HEAD %H on %ad' HEAD~..HEAD | head -n 1)\""

clean:
	rm -f check-soa
