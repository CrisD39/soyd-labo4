
APP_NAME := github.com/soyd-labo4/CrisD39


CMD_DIR := ./cmd/namenode


BIN_DIR := ./bin

.PHONY: all build run clean

all: build

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) $(CMD_DIR)

run: build
	$(BIN_DIR)/$(APP_NAME)

clean:
	rm -rf $(BIN_DIR)
