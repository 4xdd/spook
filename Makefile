MUSIC_DIR ?= $(HOME)/Music
ADDR      ?= :8080
API_URL   = http://localhost$(ADDR)

# Vite 6+ needs Node 18; Vite 8 needs Node 20+.
MIN_NODE  = 18

VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    = -s -w \
	-X github.com/spook/server/internal/version.Version=$(VERSION) \
	-X github.com/spook/server/internal/version.Commit=$(COMMIT) \
	-X github.com/spook/server/internal/version.BuildDate=$(BUILD_DATE)

RELEASE_DIR = dist/releases

.PHONY: help build build-ui build-server build-ort run dev dev-server dev-ui test clean check-node check-port release package-release convert-mert convert-mert-onnx install-ort

help:
	@echo "make build              Build the web UI and the server binary (ONNX Runtime if available)"
	@echo "make run                Build everything, then serve MUSIC_DIR=$(MUSIC_DIR)"
	@echo "make release            Cross-compile release binaries → $(RELEASE_DIR)/"
	@echo "make package-release    Build release binaries and .tar.gz / .zip archives"
	@echo "make dev                Run the Go server and the Vite dev server together"
	@echo "make test               Run the Go test suite"
	@echo "make convert-mert       Download and convert MERT-v1-95M to ~/.local/share/spook/models/"
	@echo "make convert-mert-onnx  Export MERT to ONNX for ONNX Runtime (~10-30x faster inference)"
	@echo "make install-ort        Download ONNX Runtime 1.26 shared library into ~/.local/share/spook/lib/"
	@echo "make build-ort          Same as make build-server (CGO + ONNX when lib present)"
	@echo "make clean              Remove build output"
	@echo ""
	@echo "Variables:"
	@echo "  MUSIC_DIR=$(MUSIC_DIR)"
	@echo "  ADDR=$(ADDR)          (use ADDR=:8081 if the port is busy)"

build: build-ui build-server

build-ui: check-node
	cd web && npm install --silent && npm run build

build-server:
	cd server && CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o ../bin/spook ./cmd/spook

# Alias — local Linux/macOS builds now include ONNX Runtime when CGO + lib are present.
build-ort: build-server

ORT_VERSION ?= 1.26.0
ORT_DIR ?= $(HOME)/.local/share/spook/lib
MERT_MODEL_DIR ?= $(HOME)/.local/share/spook/models

install-ort:
	@mkdir -p $(ORT_DIR)
	@if [ ! -f $(ORT_DIR)/libonnxruntime.so ]; then \
		tmp=$$(mktemp -d); \
		curl -sL "https://github.com/microsoft/onnxruntime/releases/download/v$(ORT_VERSION)/onnxruntime-linux-x64-$(ORT_VERSION).tgz" | tar xz -C $$tmp; \
		cp $$tmp/onnxruntime-linux-x64-$(ORT_VERSION)/lib/libonnxruntime.so* $(ORT_DIR)/; \
		ln -sf libonnxruntime.so.$(ORT_VERSION) $(ORT_DIR)/libonnxruntime.so; \
		rm -rf $$tmp; \
		echo "ONNX Runtime $(ORT_VERSION) → $(ORT_DIR)"; \
	else \
		echo "ONNX Runtime already installed at $(ORT_DIR)"; \
	fi

convert-mert-onnx:
	@mkdir -p /tmp/mert-work
	@if [ ! -d /tmp/mert-work/venv-onnx ]; then python3 -m venv /tmp/mert-work/venv-onnx && /tmp/mert-work/venv-onnx/bin/pip install -q torch transformers onnx onnxscript huggingface_hub; fi
	@if [ ! -f /tmp/mert-work/MERT-v1-95M/pytorch_model.bin ]; then \
		/tmp/mert-work/venv-onnx/bin/python3 -c "from huggingface_hub import hf_hub_download; hf_hub_download('m-a-p/MERT-v1-95M','pytorch_model.bin',local_dir='/tmp/mert-work/MERT-v1-95M'); hf_hub_download('m-a-p/MERT-v1-95M','config.json',local_dir='/tmp/mert-work/MERT-v1-95M')"; \
	fi
	/tmp/mert-work/venv-onnx/bin/python3 scripts/export_mert_onnx.py \
		--input /tmp/mert-work/MERT-v1-95M \
		--output $(MERT_MODEL_DIR)/mert-v1-95m.onnx
	@echo "ONNX model: $(MERT_MODEL_DIR)/mert-v1-95m.onnx (+ .onnx.data)"

release: build-ui
	@mkdir -p $(RELEASE_DIR)
	@set -e; \
	for spec in \
		"linux amd64 spook-linux-amd64" \
		"linux arm64 spook-linux-arm64" \
		"darwin amd64 spook-darwin-amd64" \
		"darwin arm64 spook-darwin-arm64" \
		"windows amd64 spook-windows-amd64.exe" \
		"windows arm64 spook-windows-arm64.exe"; do \
		set -- $$spec; \
		echo "building $$3 ($$1/$$2)"; \
		cd server && CGO_ENABLED=0 GOOS=$$1 GOARCH=$$2 go build -ldflags "$(LDFLAGS)" -o ../$(RELEASE_DIR)/$$3 ./cmd/spook; \
		cd ..; \
	done

package-release: release
	@set -e; \
	for spec in \
		"linux-amd64 spook-linux-amd64 spook tar.gz" \
		"linux-arm64 spook-linux-arm64 spook tar.gz" \
		"darwin-amd64 spook-darwin-amd64 spook tar.gz" \
		"darwin-arm64 spook-darwin-arm64 spook tar.gz" \
		"windows-amd64 spook-windows-amd64.exe spook.exe zip" \
		"windows-arm64 spook-windows-arm64.exe spook.exe zip"; do \
		set -- $$spec; \
		name="spook-$(VERSION)-$$1"; \
		staging="$(RELEASE_DIR)/$$name"; \
		rm -rf "$$staging"; \
		mkdir -p "$$staging"; \
		cp "$(RELEASE_DIR)/$$2" "$$staging/$$3"; \
		cp .env.example "$$staging/.env.example"; \
		cp scripts/release-install.txt "$$staging/INSTALL.txt"; \
		if [ "$$4" = "tar.gz" ]; then \
			tar -C "$(RELEASE_DIR)" -czf "$(RELEASE_DIR)/$$name.tar.gz" "$$name"; \
		else \
			(cd "$(RELEASE_DIR)" && zip -rq "$$name.zip" "$$name"); \
		fi; \
		rm -rf "$$staging"; \
		echo "packaged $(RELEASE_DIR)/$$name.$$4"; \
	done

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

# Requires python3 venv with torch + huggingface_hub (created on first run).
convert-mert:
	@mkdir -p /tmp/mert-work
	@if [ ! -d /tmp/mert-work/venv ]; then python3 -m venv /tmp/mert-work/venv && /tmp/mert-work/venv/bin/pip install -q torch huggingface_hub safetensors numpy; fi
	@if [ ! -f /tmp/mert-work/MERT-v1-95M/pytorch_model.bin ]; then \
		/tmp/mert-work/venv/bin/python3 -c "from huggingface_hub import hf_hub_download; hf_hub_download('m-a-p/MERT-v1-95M','pytorch_model.bin',local_dir='/tmp/mert-work/MERT-v1-95M'); hf_hub_download('m-a-p/MERT-v1-95M','config.json',local_dir='/tmp/mert-work/MERT-v1-95M')"; \
	fi
	/tmp/mert-work/venv/bin/python3 scripts/convert_mert.py \
		--input /tmp/mert-work/MERT-v1-95M \
		--output $(MERT_MODEL_DIR)/mert-v1-95m.mert
	@echo "MERT weights: $(MERT_MODEL_DIR)/mert-v1-95m.mert"

clean:
	rm -rf bin dist server/internal/web/dist/assets web/node_modules/.vite
