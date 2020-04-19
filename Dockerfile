
FROM golang:1.14-stretch AS build

# Копируем исходный код в Docker-контейнер
ADD golang/ /opt/build/golang/

# Собираем генераторы
WORKDIR /opt/build/golang
RUN go build main.go

FROM ubuntu:18.04

MAINTAINER Andrey Romanov

# Обвновление списка пакетов
RUN apt-get -y update

ENV PGVER 10
RUN apt-get install -y postgresql-$PGVER

# Run the rest of the commands as the ``postgres`` user created by the ``postgres-$PGVER`` package when it was ``apt-get installed``
USER postgres

# Create a PostgreSQL role named ``docker`` with ``docker`` as the password and
# then create a database `docker` owned by the ``docker`` role.
RUN /etc/init.d/postgresql start &&\
    psql --command "CREATE USER docker WITH SUPERUSER PASSWORD 'docker';" &&\
    createdb -O docker docker --locale='C.UTF-8' &&\
    /etc/init.d/postgresql stop

# Adjust PostgreSQL configuration so that remote connections to the
# database are possible.
RUN echo "host all  all    0.0.0.0/0  md5" >> /etc/postgresql/$PGVER/main/pg_hba.conf

# And add ``listen_addresses`` to ``/etc/postgresql/$PGVER/main/postgresql.conf``
RUN echo "listen_addresses='*'\nsynchronous_commit = off\nfsync = off\nshared_buffers = 512MB\neffective_cache_size = 1024MB\nfull_page_writes = off" >> /etc/postgresql/$PGVER/main/postgresql.conf

# Expose the PostgreSQL port
EXPOSE 5432

# Add VOLUMEs to allow backup of config, logs and databases
VOLUME  ["/etc/postgresql", "/var/log/postgresql", "/var/lib/postgresql"]

# Back to the root user
USER root

# Установка golang
RUN apt-get install -y git


# Собираем генераторы
COPY --from=build /opt/build/golang/main /usr/bin/
EXPOSE 5000

WORKDIR /opt/build/golang
# Запускаем PostgreSQL и сервер
#
CMD service postgresql start && /usr/bin/main
