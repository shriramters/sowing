all: build

build: web-assets
	go build -o sowing ./cmd/sowing

web-assets:
	@echo "Preparing web assets..."
	@mkdir -p internal/web/static/bootstrap
	@cd internal/web && npm install
	@cp internal/web/node_modules/bootstrap/dist/css/bootstrap.min.css internal/web/static/bootstrap/
	@cp internal/web/node_modules/bootstrap/dist/js/bootstrap.bundle.min.js internal/web/static/bootstrap/

clean:
	@echo "Cleaning up..."
	@rm -f sowing sowing.db
	@rm -rf internal/web/static/bootstrap
	@cd internal/web && rm -rf node_modules package-lock.json

.PHONY: all build clean web-assets
