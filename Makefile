REPO := cmsn-rotate
NAME   := 412530390400.dkr.ecr.ap-northeast-2.amazonaws.com/${REPO}
TAG    := $$(git log -1 --pretty=%h)
IMG    := ${NAME}:${TAG}
LATEST := ${NAME}:latest
DIFF   := $$(git status --porcelain)
 
# HELP
# This will output the help for each task
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help

help:
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help

## https://stackoverflow.com/questions/3878624/how-do-i-programmatically-determine-if-there-are-uncommitted-changes
## 참고용으로 남겨뒀고 현재 사용안함
diff:
	@echo $( if [ -n "$$(git status --porcelain)" ]; then \
		echo "there are changes"; \
		false; \
	else \
		echo "tree is clean"; \
		true; \
	fi )

## git에 diff가 있는지 확인하고 있으면 exit 1 반환
## || 기호는 앞에서부터 순차적으로 실행하되, 명령 실행에 성공하면 뒤에 오는 명령을 실행하지 않는다. | 기호는 둘을 이어서 실행하는 것, &&는 무조건 둘 다 실행
## echo -e \033[47;31m 어쩌구 하는 부분은 출력 메시지 색상 지정
checkdiff:
	@(git diff --exit-code && git diff --cached --exit-code) || ( echo -e "\033[47;31m Git tree is dirty! Commit first. \033[0m" && exit 1 )

echo:
	@echo "run!"

show-commit: ## Show a commit by hash. ex) make show-commit HASH=1q2w3e
	@git show $(HASH)

show-commit-list: ## Show pretty commit list.
	@git log --pretty=format:"%h - %an, %ar : %s"

login: ## ECR login.
	@rm ~/.docker/config.json && aws ecr get-login-password --region ap-northeast-2 | docker login --username AWS --password-stdin 412530390400.dkr.ecr.ap-northeast-2.amazonaws.com

build: ## Build the container and tag with git commit's hash.
	@docker build --build-arg COMMIT_HASH=$(git rev-parse --short HEAD) -t ${IMG} .
	@docker tag ${IMG} ${LATEST}
 
push: checkdiff ## Push docker image to ecr repo.
	@docker push ${IMG}
	@docker push ${LATEST}

all: checkdiff login build push ## All publish process.

run: 
	@. ./.env.sh && \
	if [ -f .last_run ]; then \
		LAST_RUN=$$(cat .last_run); \
		NEXT_RUN=$$((LAST_RUN + 1)); \
	else \
		NEXT_RUN=1; \
	fi && \
	echo $$NEXT_RUN > .last_run && \
	go run ./arg $$NEXT_RUN