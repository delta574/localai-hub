.PHONY: all dev build-frontend build build-all clean run

all: build

dev-frontend:
	cd web && npm run dev

dev-backend:
	go run -tags dev .

dev:
	start cmd /c "cd web && npm run dev"
	start cmd /c "$(MAKE) dev-backend"

build-frontend:
	cd web && npm ci && npm run build

build: build-frontend
	go build -ldflags="-s -w" -o dist/LocalAI.exe .

build-all: build-frontend
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/LocalAI-win-x64.exe .
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/LocalAI-linux-x64 .
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/LocalAI-mac-x64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/LocalAI-mac-arm64 .

run: build
	dist/LocalAI.exe --port 8080

installer: build
	"C:/Program Files (x86)/NSIS/makensis.exe" installer/installer.nsi

clean:
	rm -rf web/build web/node_modules dist/LocalAI*.exe
