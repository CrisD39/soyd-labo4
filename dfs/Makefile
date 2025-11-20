BIN_DIR := ./bin

APP_NAMENODE := namenode
APP_CLIENT   := dfsclient
APP_DATANODE := datanode

CMD_NAMENODE := ./cmd/namenode
CMD_CLIENT   := ./cmd/client
CMD_DATANODE := ./cmd/datanode

.PHONY: all build-namenode build-client build-datanode \
        run-namenode run-client run-datanode clean

all: build-namenode build-client build-datanode

build-namenode:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAMENODE) $(CMD_NAMENODE)

build-client:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_CLIENT) $(CMD_CLIENT)

build-datanode:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_DATANODE) $(CMD_DATANODE)

run-namenode: build-namenode
	$(BIN_DIR)/$(APP_NAMENODE)

run-client: build-client
	$(BIN_DIR)/$(APP_CLIENT) $(ARGS)

run-datanode: build-datanode
	$(BIN_DIR)/$(APP_DATANODE)

clean:
	rm -rf $(BIN_DIR)
