### Kubebuilder commands use to set up the project

```
mkdir second-operator
cd second-operator
kubebuilder init --domain arshsharma.com --repo github.com/RinkiyaKeDad/second-operator
kubebuilder create api --group batch --version v1 --kind PostgresWriter --resource true --controller true --namespaced true
```


### Basic commands


Command to get the IP Address:

```
ifconfig -u | grep 'inet ' | grep -v 127.0.0.1 | cut -d\  -f2 | head -1
```

```
kind create cluster
docker run -d -p 5432:5432 --name postgres -e POSTGRES_PASSWORD=password postgres
psql -h [IP ADDRESS] -p 5432 -U postgres
```

In a new terminal export these variables then:


```
export PSQL_HOST=[IP ADDRESS]
export PSQL_PORT=5432
export PSQL_DBNAME=postgres
export PSQL_USER=postgres
export PSQL_PASSWORD=password
```

In the same terminal run:

```
make install
make run
```

Then apply the YAMLs in a different terminal:

```
k apply -f config/samples/batch_v1_postgreswriter.yaml
k delete -f config/samples/batch_v1_postgreswriter.yaml
```


### Docker commands
```
docker container stop postgres
docker container ls
docker container prune
```

### Postgres specific commands

```
\l -> list all databases
\d -> list all tables
```

```
CREATE TABLE students (
	id VARCHAR(50) PRIMARY KEY,
	name VARCHAR(50),
	age INT,
	country VARCHAR(50));

select * from students;

drop table students;
```