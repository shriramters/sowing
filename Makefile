all: build

build: web-assets
	go build -o sowing ./cmd/sowing

web-assets:
	@echo "Preparing web assets..."
	@mkdir -p internal/web/static/bootstrap
	@mkdir -p internal/web/static/bootstrap-icons/font
	@mkdir -p internal/web/static/editor
	@mkdir -p internal/web/static/fonts
	@cd internal/web && npm install
	@cp internal/web/node_modules/bootstrap/dist/css/bootstrap.min.css internal/web/static/bootstrap/
	@cp internal/web/node_modules/bootstrap/dist/js/bootstrap.bundle.min.js internal/web/static/bootstrap/
	@cp -r internal/web/node_modules/bootstrap-icons/font internal/web/static/bootstrap-icons/
	@echo "Bundling CodeMirror assets with esbuild..."
	@cd internal/web && npx esbuild editor-source.js --bundle --outfile=static/editor/bundle.js --minify

clean:
	@echo "Cleaning up..."
	@rm -f sowing sowing.db
	@rm -rf internal/web/static/bootstrap
	@rm -rf internal/web/static/editor
	@cd internal/web && rm -rf node_modules package-lock.json

.PHONY: all build clean web-assets
