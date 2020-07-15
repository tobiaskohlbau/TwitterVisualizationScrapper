FROM golang AS builder

COPY . /app

WORKDIR /app
RUN go build .

FROM ubuntu

COPY --from=builder /app/TwitterVisualizationScrapper /twitter

CMD [ "/twitter" ]

