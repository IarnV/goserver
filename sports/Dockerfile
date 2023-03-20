FROM golang

WORKDIR /app

COPY . ./

RUN go mod download

RUN go build -o /sports

EXPOSE 8080

CMD [ "/sports" ]