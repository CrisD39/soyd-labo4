BIN_DIR := ./bin

APP_NAMENODE := namenode
APP_CLIENT   := dfsclient
APP_DATANODE := datanode
APP_LOG      := logger
APP_WEBCLIENT := webclient

DATANODE_ID ?= d2



CMD_NAMENODE := ./cmd/namenode
CMD_CLIENT   := ./cmd/client
CMD_DATANODE := ./cmd/datanode
CMD_LOG      := ./cmd/logger
CMD_WEBCLIENT := ./cmd/webclient

.PHONY: all \
        build-namenode build-client build-datanode build-log build-webclient \
        run-namenode run-client run-datanode run-log run-webclient \
        clean

# Compila todo
all: build-namenode build-client build-datanode build-log build-webclient

build-namenode:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAMENODE) $(CMD_NAMENODE)

build-client:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_CLIENT) $(CMD_CLIENT)

build-datanode:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_DATANODE) $(CMD_DATANODE)

build-log:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_LOG) $(CMD_LOG)

build-webclient:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_WEBCLIENT) $(CMD_WEBCLIENT)

run-namenode: build-namenode
	$(BIN_DIR)/$(APP_NAMENODE)

run-client: build-client
	$(BIN_DIR)/$(APP_CLIENT) $(ARGS)

run-datanode: build-datanode
	$(BIN_DIR)/$(APP_DATANODE) $(DATANODE_ID)

run-log: build-log
	$(BIN_DIR)/$(APP_LOG)

run-webclient: build-webclient
	$(BIN_DIR)/$(APP_WEBCLIENT)

clean:
	rm -rf $(BIN_DIR)
