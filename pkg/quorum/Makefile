all: test-quorum

test-quorum:
	${GODEP} go test ./...  -logtostderr -check.vv -v ${TEST_ARGS}
