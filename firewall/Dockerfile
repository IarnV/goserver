FROM golang

WORKDIR /app

COPY . ./

RUN go mod download

RUN go build -o /firewall

EXPOSE 8081

CMD [ "/firewall" ]
