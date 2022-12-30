GO?=go

# test 单元测试 和 代码覆盖率
.PHONY: test
test:
	$(GO) test -coverprofile=./test/cover.out
	$(GO) tool cover -html=./test/cover.out -o ./test/coverage.html
	open ./test/coverage.html