SHORT=0
FLAGS=-v -failfast
ifeq ($(SHORT),1)
TESTFLAGS :=
else
TESTFLAGS := -short
endif

qsub/qsub: qsub/*.go
	go build -o $@ $^

experiment:
	go build . && scp -C pbqff 'woods:.'

cover:
	go test . -v -coverprofile=/tmp/pbqff.out; go tool cover -html /tmp/pbqff.out

profcart:
	go test . -v -short -run '^TestCart$$' -cpuprofile=/tmp/cart.prof

deploy: pbqff
	scp -C pbqff 'woods:Programs/pbqff/.'

beta: pbqff
	scp -C pbqff 'woods:Programs/pbqff/beta/.'

alpha: pbqff
	scp -C pbqff 'woods:Programs/pbqff/alpha/.'

test: qsub/qsub version.go
	go test . $(TESTFLAGS) $(FLAGS)

bench: qsub
	go test . $(TESTFLAGS) -bench 'CheckLog|CheckProg|ReadOut'

docs:
	scp -r tutorial/main.pdf 'woods:Programs/pbqff/docs/tutorial.pdf'
	scp -r manual/pbqff.1 'woods:Programs/pbqff/docs/man1/.'

clean:
	rm -f tests/cart/cart.err tests/cart/cart.out
	rm -f tests/grad/grad.err tests/grad/grad.out
	rm -f tests/sic/sic.err tests/sic/sic.out
	rm -rf tests/cart/pts
	rm -rf tests/grad/pts
	rm -rf tests/sic/opt tests/sic/freq tests/sic/freqs tests/sic/pts
	rm -rf tests/cart/fort.* tests/cart/spectro.out

pbqff: *.go version.go
	go build .

version.go: .git
	./scripts/version.pl

eland: pbqff
	scp -C pbqff 'eland:programs/pbqff/.'
