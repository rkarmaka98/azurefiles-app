# AZFiles-Anomaly Demo

This project is a comprehensive local end-to-end demonstration showcasing the power and flexibility of Azure Files, encompassing provisioning, real-time metrics collection, anomaly detection, and an intuitive vanilla JavaScript dashboard—all running seamlessly on your personal workstation without relying on Kubernetes or complex cloud-native stacks.

## High-Level Architecture

```
+-------------+           +----------------------+           +---------------+
|             |  HTTPS    |                      |  ARM API  |               |
| Web Browser +<--------->+ Go HTTP API (/api)   +<--------->+ Azure Monitor |
| (Dashboard) |           | (main.go + monitor)  |           |  (Metrics)    |
+------+------+
       ^     |
       |     | Poll every minute for JSON
       v     |
+------+------+
|             |
| CLI Commands|
| (azfilesctl)|
+-------------+
```

1. **`azfilesctl serve`** starts two concurrent components:

   * **Monitor Loop (`runMonitorLoop`)**:

     * Authenticates via **DefaultAzureCredential**.
     * Fetches four critical metrics per share: **IOPS**, **Bandwidth**, **Latency**, and **Transactions**.
     * Filters results to individual shares via the **FileShare** dimension.
     * Applies a **z-score detector** to flag anomalies (default threshold >3σ).
     * Stores anomaly alerts in a thread-safe in-memory map.
   * **HTTP API (`serveAPI`)**:

     * Serves **CORS-enabled** endpoints on **port 8080**:

       * **`GET /api/shares`**: returns JSON array of each share’s **name**, **quota**, and current metric values.
       * **`GET /api/anomalies`**: returns JSON map of share name to alert message for any detected anomalies.
2. **Web Dashboard (index.html + script.js)**:

   * Hosted on **port 8000** (or any static server).
   * Polls the `/api/shares` and `/api/anomalies` endpoints every minute.
   * Renders a responsive HTML table with seven columns: **Name**, **Quota (GB)**, **IOPS**, **Bandwidth (MiB/s)**, **Latency (ms)**, **Transactions/s**, and **Status**.
   * Dynamically highlights rows with a **⚠** when anomalies are present.
3. **Load Simulation**:

   * Uses **HTTPS-based** operations (AzCopy or Azure CLI) to generate real traffic spikes on the file share endpoints.
   * Bypasses the need for SMB/mounting and port 445, ideal for restricted environments.

---

## Features

* **Go CLI (`azfilesctl`)**:

  * **Scaffolded with Cobra**, offering `create`, `list`, `delete`, and `anomalies` commands.
  * **Environment-based credentials**, reading from `AZURE_*` variables.
* **Anomaly Detection**:

  * **Z-score algorithm** with a configurable window (default 20 samples) and threshold.
  * **Pluggable metric definitions**, easily add or remove metrics in one place.
* **Metrics Collected**:

  1. **FileShareMaxUsedIOPS** – spike detection in IOPS usage.
  2. **FileShareMaxUsedBandwidthMiBps** – detection of throughput surges.
  3. **SuccessE2ELatency** – capturing client-perceived performance degradation.
  4. **Transactions** – identifying bursts of high-frequency operations.
* **HTTP API** & **CORS**:

  * Lightweight server using Go’s `net/http` package.
  * Cross-origin requests enabled for seamless integration with static frontends.
* **Vanilla JavaScript Dashboard**:

  * No frameworks required: just DOM manipulation and `fetch` API.
  * Real-time refresh and user feedback.

---

## Prerequisites

* **Go 1.19+** (development and CLI runtime).
* **Azure CLI** (authenticated via `az login`).
* **AzCopy v10** (for high-throughput load testing).
* **Python 3.9+** (optional; serves static files with `http.server`).
* **Storage Account** (`staccazfilesdemo`) with at least two file shares: `share1`, `share2`.
* **Service Principal** (`AZFILES_APP`) or user with **Storage File Data SMB Share Contributor** and **Monitoring Reader** roles.

---

## Setup & Configuration

### 1. Set Environment Variables

```bash
export AZURE_TENANT_ID="<your-tenant-id>"
export AZURE_CLIENT_ID="<your-client-id>"
export AZURE_CLIENT_SECRET="<your-client-secret>"
export AZURE_SUBSCRIPTION_ID=$(az account show --query id -o tsv)
export AZURE_RESOURCE_GROUP="rg-azfiles-demo"
export AZURE_STORAGE_ACCOUNT="staccazfilesdemo"
export AZURE_STORAGE_KEY=$(az storage account keys list \
  --resource-group $AZURE_RESOURCE_GROUP \
  --account-name $AZURE_STORAGE_ACCOUNT \
  --query "[0].value" -o tsv)
```

### 2. Derive File Service Resource ID

```bash
FILE_SVC_ID=$(az resource show \
  --resource-group $AZURE_RESOURCE_GROUP \
  --namespace Microsoft.Storage \
  --resource-type storageAccounts/fileServices \
  --name "$AZURE_STORAGE_ACCOUNT/default" \
  --query id -o tsv)
echo "File Service ID: $FILE_SVC_ID"
```

This ID is the scope for all monitor API calls.

---

## Building & Running

### 1. Build the Go CLI

```bash
cd azfiles-demo/cli
go mod tidy
go build -o azfilesctl
```

### 2. Start Monitoring & API Server

```bash
./azfilesctl \
  --subscription "$AZURE_SUBSCRIPTION_ID" \
  --shares "share1:${FILE_SVC_ID},share2:${FILE_SVC_ID}" \
  serve
```

Expect:

```
📡 JSON API listening on :8080
```

### 3. Launch the Dashboard

Option A: Python HTTP server

```bash
cd azfiles-demo/web
python3 -m http.server 8000
open http://localhost:8000/index.html
```

Option B: Simple Node server

```bash
npm install -g serve
cd azfiles-demo/web
serve -l 8000
```

---

## Simulating Load & Triggering Anomalies

### A) Bulk Upload with AzCopy

```bash
# Generate many small files
mkdir -p /tmp/spike && \
for i in $(seq 1 2000); do \
  dd if=/dev/urandom of=/tmp/spike/file_$i bs=4K count=1; \
done

# Upload recursively to share1
azcopy copy "/tmp/spike/*" \
  "https://${AZURE_STORAGE_ACCOUNT}.file.core.windows.net/share1" \
  --recursive --overwrite=true
```

This will drive high IOPS and transactions, observable within one polling cycle.

### B) Azure CLI Loop for Throughput

```bash
# Single large file
dd if=/dev/zero of=/tmp/large.bin bs=1M count=500
for i in {1..20}; do \
  az storage file upload \
    --account-name $AZURE_STORAGE_ACCOUNT \
    --account-key $AZURE_STORAGE_KEY \
    --share-name share1 \
    --source /tmp/large.bin \
    --path large_$i.bin &
  sleep 0.5

done; wait
```

Generates sustained bandwidth consumption spikes.

Refresh the dashboard after \~1 minute to see colored alerts.

---

## Testing & Validation

1. **HTTP API**:

   * `curl -s http://localhost:8080/api/shares | jq .` should list each share with all four metric values.
   * `curl -s http://localhost:8080/api/anomalies | jq .` shows any active alerts.
2. **Dashboard**: Ensure rows turn red with the ⚠ icon next to metric spikes.
3. **Manual Z-score Tuning**:

   * In `detector.go`, modify `>3.0` to `>1.5` for a lower threshold.

---

## Cleanup

```bash
# Stop CLI process (Ctrl+C)
# Stop static server
# Remove generated spike files
rm -rf /tmp/spike
```

---

## Next Steps & Extensions

* **Add more dimensions**: monitor per-API or per-geo metrics.
* **Persist state**: push anomalies to Azure Table or CosmosDB.
* **Notifications**: integrate webhooks or messaging via Azure Functions.
* **Containerization**: build a Docker image for `azfilesctl` and host the dashboard.
* **Cloud Deployment**: deploy on Azure App Service or AKS for real-world usage.

Enjoy this fully self-contained Azure Files anomaly monitoring demo! Feel free to extend and share your improvements.
