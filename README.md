# mposter

mposter is a tool to HTTP-POST massive amounts of data one row at a time

Think of `cat ids.list | while read id; do echo ${id}; curl --fail -XPOST http://host:port/post/${id}; done` with extra conveniences: 

* dry run
* greppable OK/ERR output
* progress reported to stderr 
* rate limiting

### Use case

A recurring task my teams face is to perform certain actions for large volumes of entities, e.g.: 

* publish historical data for a newly created data interface
* correct some entities
* etc. 

While these goals can often be achieved in many ways (a massive update SQL, a map reduce job on your distributed computation network to name the few), we find it often convenient to expose a simple REST POST endpoint to accept an identifier of an entity to be processed, and orchestrate the process from outside of our service. 

# Usage and features

## input

Given a file `ids.list` containing well, IDS: 

````
$ cat ids.list
1
A
3fa#g
...
````

The following command would sequentually post to `http://host:port/path/1`, `http://host:port/path/A`, `http://host:port/path/3fla%32g`, ... :

````
$ cat ids.list | mposter http://host:port/path/
````
OR 

````
$ mposter http://host:port/path/ --input=ids.list
````

## input --separator 

````
$ echo a,b | mposter http://host:port/path{{0}}/sub/{{1}}  --separator=","
````

is equivalent to 

````
$ echo a b | mposter http://host:port/path{{0}}/sub/{{1}}
````

## input --skip-line

Allows to skip the column names header by setting it to 1 or 2 from the default 0. Or maybe you want to continue from certain point.

## url 

By default, the sole value from the input line is added to the url provided. Placeholders allow for more flexible URL structures: 

````
$ echo a,b | mposter http://host:port/path/{{0}}/subpath/{{1}}
````

would produce a call to `http://host:port/path/a/subpath/b`

### HTTPS 

Supported

### Authorization 

Not supported. Might be added in the future as a header/cookie parameter. For now, consider a simple [proxy](https://golang.org/pkg/net/http/httputil/#NewSingleHostReverseProxy) taking care of this concern.

## Rate limiting

`--minimal-duration=5s` would enforce a delay of at least 5 seconds between the start of the consequent calls. 

## Parallelism 

The calls are peformed strictly consequal. Next call is made as soon as the previous finished, unless the rate limiting above kicks in.

## --tick 100

Prints the status every 100 lines (default) to stderr. Set to 0 to disable.

## --output

Every input line would be printed followed by `OK` for http 2xx response from the target URL or `ERR` and some description of this error if the call wasn't that succesfull. 

`--output` defaults to `-`, which means stdout. 

## --stop-on-initial-error 1

A number of initial calls failed to abort the run. No effect whatsover after receving a HTTP 2xx response.

## --stop-on-error 0

If set to a number different from 0 would stop the run upon receiving a given number of consequitive call failures.

## --stop-on-http-code 4xx

Comma separated list of http codes to abort the run immediately upon receiving. Comma separated. 4xx means all starting with 4. 

## circuit breaker

Not implemented, but thought of. Would look along the lines of `--break-circuit-open-on-count=10 --break-circuit-delay=1m`
