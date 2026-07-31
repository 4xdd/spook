MUSIC_DIR ?= $(HOME)/Music
ADDR      ?= :8080
API_URL   = http://localhost$(ADDR)

# Vite 6+ needs Node 18; Vite 8 needs Node 20+.
MIN_NODE  = 18

.PHONY: help build build-ui build-server run dev dev-server dev-ui test clean check-node check-port

help:
	@echo "make build              Build the web UI and the server binary"
	@echo "make run                Build everything, then serve MUSIC_DIR=$(MUSIC_DIR)"
	@echo "make dev                Run the Go server and the Vite dev server together"
	@echo "make test               Run the Go test suite"
	@echo "make clean              Remove build output"
	@echo ""
	@echo "Variables:"
	@echo "  MUSIC_DIR=$(MUSIC_DIR)"
	@echo "  ADDR=$(ADDR)          (use ADDR=:8081 if the port is busy)"

build: build-ui build-server

build-ui: check-node
	cd web && npm install --silent && npm run build

build-server:
	cd server && go build -o ../bin/spook ./cmd/spook

run: build
	./bin/spook -music-dir "$(MUSIC_DIR)" -addr "$(ADDR)" -open

check-node:
	@node -e "const v=process.versions.node.split('.').map(Number); const ok=v[0]>=$(MIN_NODE); if(!ok){console.error('Node '+process.versions.node+' is too old. Need Node $(MIN_NODE)+.'); process.exit(1)}"

# Fail early with a useful hint instead of a cryptic bind error.
check-port:
	@port="$(ADDR)"; port="$${port##*:}"; \
	if ss -tln 2>/dev/null | grep -q ":$$port "; then \
		echo "Port $(ADDR) is already in use."; \
		echo "Free it:  fuser -k $$port/tcp"; \
		echo "Or use:   make dev ADDR=:8081"; \
		exit 1; \
	fi

# Serves the UI from disk; rebuild with `make build-ui` to pick up frontend changes
# without restarting Go.
dev-server: check-port
	cd server && go run ./cmd/spook -music-dir "$(MUSIC_DIR)" -addr "$(ADDR)" -web-dir internal/web/dist

dev-ui: check-node
	cd web && SPOOK_API="$(API_URL)" npm run dev

dev:
	@$(MAKE) -j2 dev-server dev-ui

test:
	cd server && go test ./...

clean:
	rm -rf bin server/internal/web/dist/assets web/node_modules/.vite
