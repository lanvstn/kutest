# Kutest

Kutest is a Ginkgo helper library to help you write tests for your Kubernetes clusters.

It aims to be easy to operate in the following ways:

- Nothing is required to be deployed on the cluster to run tests.
- Readable test code that cuts down on the noise and gets to the point of the test.
- You can run the suite from your local machine or from the cluster depending on your use-case. It works the same. This brings you a nice local development loop to iterate on changes without the logic being too different in a continuous integration context.
- Parallelism-friendly to speed up test suite performance.

Here's how it looks:

```go
Specify("hello world", func() {
    WithJob(JobOptions{Namespace: "default"}, func() {
        fmt.Fprintln(GinkgoWriter, "hello from pod!")
    })
})
```

Read [kutest_test.go](./kutest_test.go) for examples, such as testing connectivity between two pods. 

## Usage

Install to your project: `go get github.com/lanvstn/kutest`

You'll want to package your test as a container image. Read [Dockerfile](./Dockerfile) for an example.

Here's a quick example with Kind:

```sh
> go install github.com/onsi/ginkgo/v2/ginkgo

> export KUTEST_IMAGE="localhost/kutest-example:latest"
> export KUTEST_DEFAULTIMAGEPULLPOLICY="IfNotPresent" # Only for Kind clusters.

> docker build . -t "$KUTEST_IMAGE" && \
    kind load docker-image "$KUTEST_IMAGE" && \
    ginkgo run --json-report=report.json -v

> go run ./cmd/kutesthtml < report.json > report.html
```

### Note on image settings

It's very important that whatever is running locally (on the _controller_) is the same 
as on the pods spawned by Kutest due to how Kutest identifies its pods (using `runtime.Callers`).
Keep this in mind when building images. 

- If `KUTEST_IMAGE` is a tagged with 
something you're planning to overwrite, don't set `KUTEST_DEFAULTIMAGEPULLPOLICY`. 
- Make sure others are not pushing the same `KUTEST_IMAGE` during your testing.

## Documentation

### Pitfalls

This is a weekend project. I can fix some of these up. But for now these are easier to document. Watch out for:

- Don't `WithJob` from inside a `WithJob`!
  - I can prevent this from being possible but have not added this safeguard
  - A `WithJob` should be able to `WithJob` in the future. One scenario is testing the running of jobs from a job (RBAC). However, currently this is not possible due to the single-controller design.

### Using with different test frameworks

You may have an exisiting test suite. For example, let's say you're doing continuous load testing using K6.

This is do-able with Kutest as well: you don't have to implement everything in Go.

1. Write a Go helper around your test tool
2. Ensure the dependencies are packaged in your test container image
3. Use your helper. E.g.:

```golang
It("passes the K6 loadtest", func() {
    WithJob(JobOptions{
        Namespace: "default",
        // TODO: may want to override some resource limits here
    }, func() {
        // TODO: implement K6Helper, which runs a local k6 binary with the given file
        K6Helper("my_loadtest.js")
    })
})
```

### About reporting

Ginkgo has a built-in report generation system. You can use its output by passing `--json-report=report.json` to `gingko run`. 

`./cmd/kutesthtml` provides a basic `html/template` coverter for this JSON format that uses stdin and stdout. 

On the resulting page you can view the result of the suite, every test, and the logs of the jobs that were started by Kutest.

### WithJob

WithJob makes your test suite replicate itself throughout your cluster. Based on the current pod name, the library is aware whether it is running inside the job where the passed function is desired to run. 

The first instance of the running test suite is called the _controller_. This is determined by the `KUTEST_SESSID` environment variable: if it is set, the current instance is part of an existing session and cannot be the controller. The controller creates the session ID and adds it through the environment variable to each job it creates.

This gives the appearance of the test function being magically sent to a new job for execution, but without actually sending the code around and all complexity that comes with it.

It is not recommended to pass anything out of your WithJob call since the function you pass to it will not always run, but the other logic in your test will!

## Project status

- Weekend project.
- Experimental library with unstable API.
- Not known to be used in production.
- Accepting contributions but make an issue before starting a PR.

---

Made with :3 by Lander