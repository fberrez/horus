# Horus
> Since Horus was said to be the sky, he was considered to also contain the sun and moon.

![horus illustration](.github/Horus.png "Horus Illustration")

## Disclaimer:
The API has only been tested with [LIFX Mini White](https://www.lifx.com/products/lifx-mini-white-e26).

## What is Horus?
Horus is an API which handles your [LIFX](https://www.lifx.com/) devices in your local network. It uses UDP packets to interact with them. It has been designed to simplify your interactions with your LIFX devices, without cloud connection.

## Getting started
### Edit your config file and your list of LIFX devices
1. Go in `config.yaml`.
2. Follow the documentation.

### Run the application
1. Install [Docker](https://docs.docker.com/install/) and [docker-compose](https://docs.docker.com/compose/install/)
2. `Optional` Edit the environment variables in the `docker-compose.yml` file.
3. Download and launch the container
```sh
$ docker-compose up -d
```
3. Read the Swagger [documentation](https://app.swaggerhub.com/apis-docs/fberrez/Horus/0.0.2).
4. Generate a new API key on `localhost:2020/unsecured/generate`
3. Use it!

### Example of curl:
```sh
# Generate a new API key
$ curl -iL -X GET 'localhost:2020/unsecured/generate'

# Get all of your LIFX devices
$ curl -iL -X GET 'localhost:2020/lights/'

# Toggle your lights
$ curl -iL -X POST -H "Content-Type:application/json" --data '{"duration":1500}' 'localhost:2020/lights/toggle?selector=all'

# Toggle your light with the UUID `33d07008-2082-4d7f-82f3-04c275b70055`
$ curl -iL -X POST -H "Content-Type:application/json" --data '{"duration":1500}' 'localhost:2020/lights/toggle?selector=uuid:33d07008-2082-4d7f-82f3-04c275b70055'

# Edit the color, the power status and the label your light called `foo`
$ curl -iL -X PUT -H "Content-type:application/json" --data '{
  "hsbk": {
    "hue": 65535,
    "saturation": 65535,
    "brightness": 10000,
    "kelvin": 9000
  },
  "duration": 1500,
  "power": "on",
  "label": "bar"
}' 'localhost:2020/lights/state?selector=label:foo'
```

## Swagger documentation
You can generate the Swagger (OpenAPI) documentation by running the application and going on:
```sh
http://localhost:2020/unsecured/openapi.json

# Or with curl
$ curl -X GET 'localhost:2020/unsecured/openapi.json'
```

## TODO list:
- Add location and group routes (/lights/group & /lights/location)
- Add API Key
- Try with other devices such as [Mini Color](https://www.lifx.com/collections/featured-products/products/lifx-mini-color-e26), [Tile Kit](https://www.lifx.com/collections/featured-products/products/lifx-tile), [Plus](https://www.lifx.com/collections/featured-products/products/lifx-plus-e26)
- Open to other brands
