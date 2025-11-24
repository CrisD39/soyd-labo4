BIN_DIR := ./bin

APP_NAMENODE := namenode
APP_CLIENT   := dfsclient
APP_DATANODE := datanode
APP_LOG      := logger

CMD_NAMENODE := ./cmd/namenode
CMD_CLIENT   := ./cmd/client
CMD_DATANODE := ./cmd/datanode
CMD_LOG      := ./cmd/logger

.PHONY: all \
        build-namenode build-client build-datanode build-log \
        run-namenode run-client run-datanode run-log \
        clean

# Compila todo
all: build-namenode build-client build-datanode build-log

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

run-namenode: build-namenode
	$(BIN_DIR)/$(APP_NAMENODE)

run-client: build-client
	$(BIN_DIR)/$(APP_CLIENT) $(ARGS)

run-datanode: build-datanode
	$(BIN_DIR)/$(APP_DATANODE)

run-log: build-log
	$(BIN_DIR)/$(APP_LOG)

clean:
	rm -rf $(BIN_DIR)
