FROM golang:1.22

WORKDIR /deps
RUN apt update
RUN apt install -y python3 python3-pip python3.11-venv

WORKDIR /app
RUN python3 -m venv venv

COPY . /app

RUN go mod download
RUN bash scripts/setup.sh
RUN go build -o dist-rl-test .

CMD ["/bin/bash"]