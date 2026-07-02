.PHONY: test go-test dashboard-test dashboard-build readiness onboarding-smoke deploy-system-validate dev-refresh

test: readiness onboarding-smoke go-test dashboard-test dashboard-build

go-test:
	go test ./...

dashboard-test:
	cd start-app && pnpm test && pnpm typecheck && pnpm lint

dashboard-build:
	cd start-app && pnpm build

readiness:
	./scripts/check-release-readiness.sh

onboarding-smoke:
	./scripts/check-onboarding-smoke.sh

deploy-system-validate:
	cd deploy-system && ./validate.sh

dev-refresh:
	./scripts/dev-refresh.sh
