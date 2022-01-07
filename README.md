# wmibeat-ohm2

## Considerations for creating this beat

***Disclaimer: Elastic have [deprecated the custom Beat creation as of 7.16](https://www.elastic.co/guide/en/beats/libbeat/master//breaking-changes-7.16.html#_custom_beats_generator_is_deprecated_in_7_16). So you won't be able to create custom Beats in the future using the generators. However, you can use existing Beats as a template.***
## Intro

Elasticsearch with its Beats ecosystem provides a robust monitoring framework for multitude of use cases. Additionally the *libbeats* library and complementary tooling allows creating custom Beats for  purposes not covered by the official Beats[^1]. This post aims to document my adventures in creating such a custom Beat for my use case and the bits of information I needed to piece together to do it.

I would like to collect monitoring information for analysis from a large number of industrial mini PCs. These are deployed at remote retail locations as part of a digital media signage (DMS) solution. In case of any problems with the devices it is not possible to directly inspect the devices. Thus, we must be able to see remotely what the problem cause was (e.g., overheating, CPU load) and in general we would like to have always a remote eye on the devices.
## Collecting metrics

The PCs are made by different manufacturers and use a diverse set of hardware components. They mainly differ in CPU and GPU, and run an embedded or IoT equivalent of Windows 7-10. We are interested in temperature (GPU, CPU and board), heartbeat and load information (CPU, GPU, memory, disk and network) and in collecting custom logs (Windows event log, media play log, etc.). Most of these can be collected via the official Beats such as *metricbeat*, *winlogbeat*, *filebeat* and *heartbeat*.

What is missing from the official Beats is temperature and power information. The problem here is two-fold. First we need to measure temperature reliably in a diverse hardware and software environment. Second, we need to report the measurements also in a reliable way.

[Open Hardware Monitor](https://openhardwaremonitor.org/) (OHM) solves the first problem as it collects not just temperature, but additional information such as load, power usage, and fan control information as well from a large set of supported hardware. It runs on all embedded and IoT Windows versions we use and supports the CPUs, GPUs and mainboard sensors in our industrial PCs, so a perfect fit. Additionally, it publishes all sensor data to WMI (Windows Management Instrumentation) [^2] using a [documented interface](http://openhardwaremonitor.org/wordpress/wp-content/uploads/2011/04/OpenHardwareMonitor-WMI.pdf).

For a short period I considered a custom version of OHM, that added the option to print the sensor data to a text file, but parsing a plain text-based report creates more issues than it solves.

So I needed a method for reporting the collected metrics to *elasticsearch*. First, I looked through the community Beats[^1] whether there is already a Beat available with the missing functionality.  I found [wmibeat](https://github.com/eskibars/wmibeat) which is able to read data from WMI.  However, there are two problems with wmibeat. First, it is not able to read the data format in which OHM publishes data. The issue here is that OHM uses a metric name - value type entry for each metric, while wmibeat expects a single wide and flat format with different fields for the different metric values. Second, the last commit to wmibeat was in 2016 and I was not able to make it report to a current version elasticsearch cluster. However it works perfectly when using *logstash* as target (i.e., in my case I was evaluating Graylog besides elasticsearch). So I needed to create my own Beat.

There is a [blog post](https://www.elastic.co/blog/build-your-own-beat) from elastic for building custom Beats, but is severely outdated (useless) and the [developer guide](https://www.elastic.co/guide/en/beats/devguide/current/index.html) is not very helpful. I found a bit more recent [blog post](https://georgebridgeman.com/posts/creating-a-custom-beat/) that is a good starting point, but I encountered some issues along the way. I will document here the steps I needed to create my custom Beat.

## Creating the Beat

For the record I am using Ubuntu Linux 20.04. The steps are as follows:

1. Install Go 1.17.5 using the [official install](https://go.dev/doc/install):

	```
	wget https://go.dev/dl/go1.17.5.linux-amd64.tar.gz
	tar -czvf go1.17.5.linux-amd64.tar.gz -C /usr/local
	ln -s /usr/local/go1.17.5 /usr/local/go
	```

	Make sure to set up PATH and GOPATH as needed.

2. Download libbeats:

	```
	go get github.com/elastic/beats
	```

	I had issues with the current version (master) when compiling, so I went through the releases in reverse order until I found a version that worked for me, this is 7.12. You need to check out in `src/github.com/elastic/beats`:

	```
	cd $GOPATH/src/github.com/elastic/beats
	git checkout 7.12
	```

3. You need to download and install `mage` as it is used by libbeat to create the boilerplate Beat code:

	```
	go get -u -d github.com/magefile/mage
	cd $GOPATH/src/github.com/magefile/mage
	go run bootstrap.go
	```

	The binary will be created in the `$GOPATH/bin` directory. Make sure it is included in `$PATH`.

4. Next, you need to install some Python modules and components. For example the module `ensurepip` is hardwired in the beats create code, but it is disabled by default on Debian/ Ubuntu. You need to install the `-venv` package to have it available:

	```
	sudo apt-get install python3 python3.8-venv
	```

	Additionally I needed the following Python modules:

	```
	python3 -m pip install pyyaml requests
	```

5. We can now start with the actual work of creating the boilerplate beats code as follows:

	```
	cd $GOPATH/src/github.com/elastic/beats
	$GOPATH/bin/mage GenerateCustomBeat
	```

	 It will ask a couple of questions, based on the responses it will put the created code in `$GOPATH/src/github.com/<AUTHORNAME>/<BEATNAME>`.

6. Next, the documentation states to run `make setup` in the new beats directory, but instead you need to run `make update`:

	```
	cd $GOPATH/src/github.com/<AUTHORNAME>/<BEATNAME>
	make update
	```

      This will generate the config file and fields files.

7. You can put your custom code in `beater/<BEATNAME>.go`, and you can compile your beat using:

	```
	mage build
	```

	Or alternatively, for example if you want to cross-compile to Windows (as in my case):

	```
	GOOS=windows GOARCH=386 go build -o wmibeat.exe main.go
	```





# Generated documentation
Welcome to {Beat}.

Ensure that this folder is at the following location:
`${GOPATH}/src/github.com/atisu/wmibeat-ohm2`

## Links

- https://georgebridgeman.com/posts/creating-a-custom-beat/

## Getting Started with {Beat}

### Requirements

* [Golang](https://golang.org/dl/) 1.7

### Init Project
To get running with {Beat} and also install the
dependencies, run the following command:

```
make setup
```

It will create a clean git history for each major step. Note that you can always rewrite the history if you wish before pushing your changes.

To push {Beat} in the git repository, run the following commands:

```
git remote set-url origin https://github.com/atisu/wmibeat-ohm2
git push origin master
```

For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).

### Build

To build the binary for {Beat} run the command below. This will generate a binary
in the same directory with the name wmibeat-ohm2.

```
make
```


### Run

To run {Beat} with debugging output enabled, run:

```
./wmibeat-ohm2 -c wmibeat-ohm2.yml -e -d "*"
```


### Test

To test {Beat}, run the following command:

```
make testsuite
```

alternatively:
```
make unit-tests
make system-tests
make integration-tests
make coverage-report
```

The test coverage is reported in the folder `./build/coverage/`

### Update

Each beat has a template for the mapping in elasticsearch and a documentation for the fields
which is automatically generated based on `fields.yml` by running the following command.

```
make update
```


### Cleanup

To clean  {Beat} source code, run the following command:

```
make fmt
```

To clean up the build directory and generated artifacts, run:

```
make clean
```


### Clone

To clone {Beat} from the git repository, run the following commands:

```
mkdir -p ${GOPATH}/src/github.com/atisu/wmibeat-ohm2
git clone https://github.com/atisu/wmibeat-ohm2 ${GOPATH}/src/github.com/atisu/wmibeat-ohm2
```


For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).


## Packaging

The beat frameworks provides tools to crosscompile and package your beat for different platforms. This requires [docker](https://www.docker.com/) and vendoring as described above. To build packages of your beat, run the following command:

```
make release
```

This will fetch and create all images required for the build process. The whole process to finish can take several minutes.
