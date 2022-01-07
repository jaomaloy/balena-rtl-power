FROM balenalib/raspberrypi3-debian-golang:latest-build AS build

WORKDIR /go/src/github.com/grtlp/app

COPY /app ./

RUN go mod init

RUN go mod download

RUN go get github.com/eclipse/paho.mqtt.golang
RUN go get github.com/joho/godotenv

RUN go build

FROM balenalib/raspberrypi3-debian-golang:build

RUN apt-get update
RUN apt-get -y install libfftw3-dev cmake libusb-1.0-0-dev git
RUN apt install netcat

COPY ./install.sh /usr/src/app/install.sh
RUN ["chmod", "+x", "/usr/src/app/install.sh"]
RUN ["/usr/src/app/install.sh"]

COPY --from=build /go/src/github.com/grtlp/app .

CMD ./app
