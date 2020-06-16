```go
package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/ejohn/go-atomic/runner"
)

func main() {
    ar := &runner.Runner{
        AtomicsFolder:"atomic-red-team/atomics",
    }
    err := ar.LoadTechniques()
    if err != nil {
        log.Panic(err)
    }
	fc := runner.FilterConfig{
		Platform:   runtime.GOOS,
		Techniques: []string{"T1016"},
	}
	techs := ar.Filter(fc)
    
    rc := runner.RunConfig{
        EnableTest: true,
    }

	for _, tech := range techs {
		for _, test := range tech.AtomicTests {
			result, err := ar.Run(test, nil, rc)
			if err != nil {
				log.Println(err)
				continue
			}
			fmt.Printf("%v\n", result)
		}
	}
}
```