# Auto API Tester

An automated API testing tool that generates and executes tests based on Swagger/OpenAPI specifications.

## Features

- Automatic test case generation from Swagger documentation
- Support for all HTTP methods (GET, POST, PUT, DELETE, etc.)
- Concurrent test execution
- Detailed test reports in multiple formats (JSON, HTML)
- Configurable test parameters and retry mechanisms
- Environment-specific configurations
- Docker support for easy deployment

## Project Structure

```
auto-api-tester/
├── cmd/
│   └── main.go
├── internal/
│   ├── parser/      # Swagger/OpenAPI parser
│   ├── executor/    # Test execution engine
│   ├── reporter/    # Test reporting system
│   └── config/      # Configuration management
├── config/
│   └── config.yaml  # Configuration file
├── reports/         # Generated test reports
├── testdata/        # Test data files
├── Dockerfile       # Docker configuration
├── go.mod
└── README.md
```

## Prerequisites

- Go 1.16 or higher
- Docker (optional, for containerized deployment)
- Access to the target API's Swagger documentation

## Installation

### Local Installation

1. Clone the repository:
```bash
git clone https://github.com/karna-nsio/auto-api-tester
cd auto-api-tester
```

2. Install dependencies:
```bash
go mod download
```

3. Configure the application:
   - Set the `AUTH_TOKEN` environment variable for API authentication
   - Modify `config/config.yaml` for your specific needs

### Docker Installation

1. Build the Docker image:
```bash
docker build -t auto-api-tester .
```

## Usage

### Local Usage

1. Generate test data template from Swagger documentation:
```bash
go run main.go generate -url <swagger-url>
```

2. Review and modify the generated template:
   - Check `testdata/testdata_template.json`

3. Run the API tests:
```bash
go run main.go
```

## Configuration

The application can be configured through environment variables and the `config.yaml` file:

```yaml
environment:
  qa:
    base_url: ""
    auth:
      type: "bearer"
      token: "${AUTH_TOKEN}"

test:
  concurrent: true
  max_workers: 5
  timeout: 30
  retry:
    attempts: 3
    delay: 1

reporting:
  format: ["html", "json"]
  output_dir: "./reports"
  detailed: true
```

## Test Data Format

The test data file (`testdata.json`) should follow this structure:

```json
{
  "endpoints": {
    "GET /api/endpoint": {
      "query_params": {
        "param1": "value1",
        "param2": "value2"
      },
      "headers": {
        "Accept": "application/json",
        "Content-Type": "application/json"
      }
    }
  }
}
```

## Reports

Test reports are generated in the `reports` directory in both JSON and HTML formats. The reports include:
- Test execution timestamp
- Total number of tests
- Number of passed/failed tests
- Detailed results for each test
- Response bodies and status codes
- Error messages (if any)

### Example Test Report

```json
{
  "Timestamp": "2025-05-28T16:17:59.2626198+05:30",
  "TotalTests": 1,
  "PassedTests": 0,
  "FailedTests": 1,
  "Duration": 0,
  "Results": [
    {
      "Endpoint": "/api/ClientMapping",
      "Method": "GET",
      "Status": 200,
      "Duration": 1005222400,
      "Error": "<nil>",
      "RequestBody": "",
      "Response": {
        "hospitalCode": "NISC",
        "id": 2,
        "typeName": "ICD10Code",
        "valuePairs": {
          "C50.911": "Malignant neoplasm of unspecified site of right female breast",
          "C50.912": "Malignant neoplasm of unspecified site of left female breast",
          "C56.1": "Malignant neoplasm of right ovary",
          "C56.2": "Malignant neoplasm of left ovary",
          "Other": "",
          "Z80.0": "Family history of malignant neoplasm of digestive organs [pancreas]",
          "Z80.3": "Family history of malignant neoplasm of breast",
          "Z80.41": "Family history of malignant neoplasm of ovary [epithelial]",
          "Z80.42": "Family history of malignant neoplasm of prostate",
          "Z83.71": "Family history of colon polyps — New",
          "Z85.07": "Personal history of malignant neoplasm of pancreas",
          "Z85.3": "Personal history of malignant neoplasm of breast",
          "Z85.43": "Personal history of malignant neoplasm of ovary",
          "Z85.46": "Personal history of malignant neoplasm of prostate",
          "Z86.010": "Personal history of colon polyps — New"
        }
      }
    }
  ]
}
```

### Example Test Template

```json
{
  "endpoints": {
    "GET /api/ClientMapping": {
      "query_params": {
        "hospitalCode": "NISC",
        "typeName": "ICD10Code"
      },
      "headers": {
        "Accept": "application/json",
        "Content-Type": "application/json"
      }
    }
  }
}