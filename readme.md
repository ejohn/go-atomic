# Go runner for atomic red team test cases

## Build
```shell script
git clone https://github.com/ejohn/go-atomic.git
cd go-atomic
go build github.com/ejohn/go-atomic/cmd/go-atomic
```

## Usage
```
Usage of go-atomic:
  -arg value
    	pass argument to test [ex foo=bar], set multiple times for different arguments
  -cleanup
    	run only cleanup
  -debug
    	show debug logs
  -dependency
    	check prerequisites and get them if needed
  -dry-run
    	build test and display what will be executed when the test is run
  -guid string
    	test case guids separated by comma
  -name string
    	name of the test to run
  -num string
    	test case number [1-N]
  -path string
    	path to atomics folder
  -prereq
    	check if prerequisites for test are met
  -run
    	run dependencies, test commands and cleanup for all tests selected
  -tech string
    	list of technique id's [ex T1002,T1003]
  -test
    	run only the test, disables dependencies and cleanup
  -timeout string
    	timeout for commands [ex 1s, 2m]
```