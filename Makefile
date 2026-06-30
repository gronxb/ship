.PHONY: test go-test dashboard-test dashboard-build readiness deploy-system-validate

test: readiness go-test dashboard-test dashboard-build

go-test:
	go test ./...

dashboard-test:
	cd start-app && pnpm test && pnpm typecheck && pnpm lint

dashboard-build:
	cd start-app && pnpm build

readiness:
	./scripts/check-release-readiness.sh

deploy-system-validate:
	cd deploy-system && ./validate.sh
