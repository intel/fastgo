.PHONY: addlicense test fuzz

addlicense:
	addlicense -c "Intel Corporation." -y "(c) 2024," -s=only  -v -l BSD-3-Clause ./

test:
	go test ./... -v  -coverprofile cover.out
	go test ./... -v -tags noasmtest

lint:
	golangci-lint run ./...

fuzz:
	go test  -run=^$$ -json -fuzz=^FuzzDeflate$$ -fuzztime=30s github.com/intel/fastgo/compress/flate/internal/deflate -v 
	go test  -run=^$$ -json -fuzz=^FuzzInflate$$ -fuzztime=30s github.com/intel/fastgo/compress/flate -v 