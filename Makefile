.PHONY: tests docs qsub molpro

ifeq ($(SHORT),1)
TESTFLAGS := -v
else
TESTFLAGS := -v -short
endif

qsub/qsub: qsub/*.go
	go build -o qsub/qsub qsub/*.go

molpro/molpro: molpro/*.go
	go build -o molpro/molpro molpro/*.go

experiment:
	go build . && scp -C pbqff 'woods:.'

cover:
	go test . -v -coverprofile=/tmp/pbqff.out; go tool cover -html /tmp/pbqff.out

profcart:
	go test . -v -short -run '^TestCart$$' -cpuprofile=/tmp/cart.prof

deploy: build
	scp -C pbqff 'woods:Programs/pbqff/.'

beta: build
	scp -C pbqff 'woods:Programs/pbqff/beta/.'

test: qsub molpro
	go test . $(TESTFLAGS)

bench: qsub molpro
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

build: *.go
	./scripts/version.pl
	go build .
