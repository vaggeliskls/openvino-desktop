# Simple Makefile for OVMS setup on Windows

.PHONY: help install export run clean get-remote-export openvino build-cli ui-assets ui-dev ui-build

UV_VERSION = 0.10.7
UV = uv.exe

help:
	@echo Available commands:
	@echo   make install            - Create venv and install dependencies
	@echo   make export             - Run model export script
	@echo   make run                - Start OVMS server
	@echo   make clean              - Remove venv
	@echo   make get-remote-export  - Download and extract latest export release
	@echo   make openvino           - Download and extract OVMS server
	@echo   make build-cli          - Build the Go CLI binary
	@echo   make ui-assets          - Download uv and copy scripts into ui/assets (run once before build)
	@echo   make ui-dev             - Run Wails UI in development mode
	@echo   make ui-build           - Build Wails UI as a Windows executable


install: 
	curl -L https://github.com/astral-sh/uv/releases/download/$(UV_VERSION)/uv-x86_64-pc-windows-msvc.zip -o uv-tmp.zip
	tar -xf uv-tmp.zip uv.exe
	del uv-tmp.zip
	./uv python install 3.12.12 --install-dir ./python
	./uv venv export --python ./python/cpython-3.12.12-windows-x86_64-none/python.exe --relocatable
	./uv pip install --python export\Scripts\python.exe -r export-model-requirements\requirements.txt

export:
	export\Scripts\python.exe export-model-requirements\export_model.py $(ARGS)

run:
	ovms\setupvars.ps1 && ovms\ovms.exe --rest_port 8000 --config_path config.json

openvino:
	-rmdir /s /q ovms
	curl -L https://github.com/openvinotoolkit/model_server/releases/download/v2026.0/ovms_windows_python_on.zip -o ovms-tmp.zip
	tar -xf ovms-tmp.zip
	del ovms-tmp.zip

build-cli:
	cd cli && go build -o ..\openvino-cli.exe .

# Populate ui/assets/ with uv.exe and export scripts (required before ui-dev or ui-build)
ui-assets:
	-mkdir ui\assets
	-mkdir ui\assets\export-model-requirements
	if not exist ui\assets\uv.exe curl -L https://github.com/astral-sh/uv/releases/download/$(UV_VERSION)/uv-x86_64-pc-windows-msvc.zip -o uv-tmp.zip && tar -xf uv-tmp.zip uv.exe && move uv.exe ui\assets\uv.exe && del uv-tmp.zip
	copy export-model-requirements\requirements.txt ui\assets\export-model-requirements\requirements.txt
	copy export-model-requirements\export_model.py ui\assets\export-model-requirements\export_model.py

ui-dev: ui-assets
	cd ui && wails dev

ui-build: ui-assets
	cd ui && wails build

clean:
	rmdir /s /q export
	rmdir /s /q python
	del uv.exe