# Kutest

Kutest is a Ginkgo helper library to help you write tests for your Kubernetes clusters.

It aims to be easy to operate in the following ways:

- Nothing is required to be deployed on the cluster to run tests.
- Readable test code that cuts down on the noise and gets to the point of the test.
- You can run the suite from your local machine or from the cluster depending on your use-case. It works the same. This brings you a nice local development loop to iterate on changes without the logic being too different in a continuous integration context.

Here's how it looks:

```go
Specify("something", func() {
    WithJob(JobOptions{
        Namespace: "default",
    }, func() {
        fmt.Println("hello from job!")
    })

    fmt.Println("we're done!")
})
```

## Usage

Install to your project: `go get github.com/lanvstn/kutest`

Then read [kutest_test.go](./kutest_test.go) for examples.

You'll want to package your test as a container image. Read [Dockerfile](./Dockerfile) for an example.

Here's a quick example with Kind:

```sh
> go install github.com/onsi/ginkgo/v2/ginkgo

> export KUTEST_IMAGE="localhost/kutest-example:latest"
> export KUTEST_DEFAULTIMAGEPULLPOLICY="IfNotPresent"

> docker build . -t "$KUTEST_IMAGE" && \
    kind load docker-image "$KUTEST_IMAGE" && \
    ginkgo run
```

## How it works

WithJob makes your test suite replicate itself throughout your cluster. Based on the current pod name, the library is aware whether it is running inside the job where the passed function is desired to run. 

The first instance of the running test suite is called the _controller_. This is determined by the `KUTEST_SESSID` environment variable: if it is set, the current instance is part of an existing session and cannot be the controller. The controller creates the session ID and adds it through the environment variable to each job it creates.

This gives the appearance of the test function being magically sent to a new job for execution, but without actually sending the code around and all complexity that comes with it.

It's important to note that this library is only intended to be used in test suites that do most of the heavy lifting _inside_ your platform, so inside jobs that are created by the suite. Any logic outside of the helpers offered by Kutest will be executed on every copy it makes of itself.

## Project status

Proof of concept. Alpha software - API subject to change!

Accepting contributions but make an issue before starting a PR.

## TODO

- Selective test execution (propagation of Ginkgo test filters)
- Log propagation (through Kubernetes API)
- Reporting tools (HTML reports)
- Generic resource helpers

---

Made with :3 by Lander