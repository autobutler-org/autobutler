# autobutler

## Usage

Before any of these, set the `LLM_AZURE_API_KEY` environment variable to your Azure OpenAI API key

### Make an LLM call

To make a call to the LLM, you can use the following command:

```shell
go run main.go chat "How much milk is in my house?"
```

### Run the backend

To serve the backend, you can use the following command:

```shell
make serve
```

### Build the backend, make an LLM call

```shell
make build
./build/backend chat "How much milk is in my house?"
```
