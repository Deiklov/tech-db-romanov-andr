FROM golang:latest
MAINTAINER Romanov Andrey <romanov408g@mail.ru>
RUN mkdir /app
ADD ./golang /app
WORKDIR /app
RUN cd golang; go mod download
RUN go build -o main .
CMD ["/app/main"]