GoDouble
===============

TestDouble framework for Go.

### Why

To learn about Go.  Sane folks should probably use gomock, testify etc...

### Stubs, Mocks, Spies and Fakes

This framework creates a TestDouble implementation of an interface which can be substituted for the real thing during
  tests. Interface methods can then individually be Stubbed, Mocked, Spied upon or Faked as required.

See the canonical sources...

http://xunitpatterns.com/Test%20Double.html

https://martinfowler.com/articles/mocksArentStubs.html


### Generating Doubles 

#### Manually
Create a struct that includes the interface and pointer to godouble.TestDouble
```go
type APIDouble struct {
    API
    *godouble.TestDouble
}

func NewAPIDouble(t godouble.T,configurators) *APIDouble {
    return &APIDouble{TestDouble: godouble.NewDouble(t,(*API)(nil))}
}
```  

Implement Interfaces methods calling godouble.Invoke and converting the return types
```go
func (d *APIDouble) SomeQuery( input string) (r Results,e error) {
    returns := godouble.Invoke(d,"SomeQuery",input)
    r, _ = returns[0].(Results)
    e, _ = returns[1].(error)
    return
}
```

#### Via go:generate
Create a generator command that uses doublegen.NewGenerator over the interface

```go
// +build doublegen

package main

func main() {
	if f, e := os.Create("example_double_test.go"); e == nil {
		defer f.Close()
		doublegen.NewGenerator((*examples.API)(nil)).GenerateDouble(f)
	} else {
		log.Fatal(e)
	}
}
```

Add go:generate tag in a test file
```go
//go:generate go run -tags doublegen doublegen/example_gen.go

func Test_Mock(t *testing.T) {
	d := NewAPIDouble(t)
    //...
}
```

Run go generate
```
$ go generate
2020/01/30 22:30:55 Generated Double for examples.API
```


### Using Doubles

#### Stubbing Methods

A stub provides specific return values for a matching call to the method.  Most useful where the return values
are the primary means by which correct operation of the system under test can be verified.

```go
package examples

import (
	. "github.com/lwoggardner/godouble" //Note the dot import which assists with readability
	"testing"
)
func Test_Stub(t *testing.T) {
    d := NewAPIDouble(t)
  
    //Stub a method that receives specific arguments, to return specific values
    d.Stub("SomeQuery").Matching(Arguments(Eql("test"))).Returning(Values(Results{"result"}, nil))

    // Exercise the system under test substituting d for the real API client
    // ...

    // Verify assertions to confirm the system under test behaves as expected with the given return values
    // ...
}
```

#### Mocking Methods

A Mock is a Stub with an up-front expectation for how many times it will be called.
Most useful when the return values of the method do not completely ensure correct functioning of the system under test,


```go
func Test_Mock(t *testing.T) {
	d := NewAPIDouble(t)
    // Verify the mock expectations are met at completion
	defer d.Verify()

	//Stub a method that receives specific arguments, returns specific values and explicitly expects to be called once
	d.Mock("SomeQuery").Matching(Arguments(Eql("test"))).Returning(Values(Results{"result"}, nil)).Expect(Exactly(3))
	d.Mock("OtherMethod").Expect(Never())

    //Exercise...
}
```

#### Spying on Methods

A Spy is a record of all calls made to a method which can be verified after exercising the system under test. 
Used similarly to Mock, but where you prefer to explicitly assert received arguments and call counts in the Verify phase
of the test.

```go
func Test_Spy(t *testing.T) {
	//Setup
	d := NewAPIDouble(t)

	spy := d.Spy("SomeQuery").Returning(Values(Results{"nothing"}, nil))

	//Exercise...

	//Verify
    spy.Expect(Twice()) //All calls
	spy.Matching(Arguments(Eql("test"))).Expect(Once()) //The subset of calls with matching args
    
}
```

#### Faking a Method

A Fake is a Spy that provides an actual implementation of the method instead of return values. Use with caution.

```go
func Test_Fake(t *testing.T) {
	//Setup
	d := NewAPIDouble(t)
	impl := func( i int, options...string) *Results {
		return &Results{Output: fmt.Sprintf("%s %d",strings.Join(options," "),i)}
	}

	spy := d.Fake("QueryWithOptions",impl)

	//Exercise...
	
	//Verify
	spy.Expect(Twice())
	spy.Matching(Arguments(Eql(10))).Expect(Once())
	
}
```

#### Argument Matchers

Used in Stubs and Mocks to Setup whether the arguments in a particular call will match the stub.

Used in Spies and Fakes to select a subset of received calls from the set of all recorded calls or to take 
  subsets of subsets
  
Simple implementations are provided. eg for deep equality

#### Return Values

Used in Stubs, Mocks and Spies to generate values from potentially successive calls to the method.

Simple implementations are provided. eg for fixed values, channel of values, randomly delayed values

#### Expectations

Used in Mocks to Setup expectation on the number of times the matching method will be called

Used in Spies and Fakes to explicitly verify the number of times the method was called.

eg Once(), Twice(), Never(), Exactly(n) AtLeast(n), AtMost(n), Between(n,m)

#### Sequences

Mocks can be setup to expect being called After another mock call (to any method of any double), or a sequence of mocks
 can be setup to verify that they execute InOrder

Spies and Fakes can select a subset of calls that were made After another subset of calls (to any method of any double).
