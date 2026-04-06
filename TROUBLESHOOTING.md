# Troubleshooting & Bug Log

This document serves as an engineering post-mortem for the bugs, errors, and pitfalls encountered during the GitOps migration and observability setup on the RHEL 9.7 K3s homelab.

---

## 1. Promtail SELinux Sandbox Rejection
**Symptom**: 
Logging in to Grafana and clicking the "Log browser" resulted in a `No options found` dropdown. 

**How to Debug**:
Check the readiness probe status and the raw logs of the Promtail collector.
```bash
# 1. Check if the Pod is stuck in 0/1 Ready state
kubectl get pods -n observability -l app.kubernetes.io/name=promtail

# 2. Check the raw logs for permission errors
kubectl logs -n observability -l app.kubernetes.io/name=promtail
```
* **Error Output**: `open /run/promtail/.positions.yamlXYZ: permission denied` and Readiness Probe HTTP 500 returning `Unable to find any logs to tail`.
* **Expected Output**: Continuous lines stating `msg="Adding target" key="/var/log/pods/..."` indicating logs are successfully ingested.

**Cause**: 
RHEL 9's strict SELinux policies were blocking the Promtail container from securely mounting the host's log directories and prevented it from saving its tracker file.

**The Fix**:
Updated `k8s/observability-stack.yaml` to enforce the `spc_t` (Super Privileged Container) SELinux context on the Pod, telling the RHEL kernel to natively drop the SELinux sandbox. Promtail was also instructed to save its tracking file to `/tmp/positions.yaml`.

---

## 2. Grafana Visualization Parse Error: "col 63: syntax error"
**Symptom**:
Executing a perfectly valid 22-character query like `{app="engine-fetcher"}` in Grafana throws a brown alert box stating `Failed to load log volume for this query / parse error at line 1, col 63...`

**How to Debug**:
Verify if the target application is actually generating logs, or if it is completely silent.
```bash
# 1. Check if the target application has generated ANY logs recently
kubectl logs -l app=engine-fetcher --since=10m
```
* **Error Output**: Blank/empty output (meaning the app has been silent). 
* **Expected Output**: A stream of recent stdout logs mirroring exactly what Grafana should display.

**Cause**:
A known visual bug in the Grafana UI's "Logs Volume" histogram builder. When you execute a query that returns **0 completely blank logs**, the backend attempts to build a derived density chart in the background which crashes and throws the error. 

**The Fix**:
No code changes required. It is completely safe to ignore.

---

## 3. GitOps Ghost Prune: "secret redis-secret not found"
**Symptom**:
The `engine-fetcher` pod perpetually crashes in a `CreateContainerConfigError` loop complaining that `redis-secret` is missing.

**How to Debug**:
Verify the physical existence of the secret in the cluster.
```bash
# 1. Look for the secret in the default namespace
kubectl get secret redis-secret
```
* **Error Output**: `Error from server (NotFound): secrets "redis-secret" not found`
* **Expected Output**: `NAME: redis-secret   TYPE: Opaque   DATA: 1   AGE: 10m`

**Cause**:
During Phase 5, we removed the plain-text `redis-secret` from Git to protect the credentials. When ArgoCD detected the secret was absent from Git, its automated `prune: true` feature aggressively deleted the secret from the cluster because ArgoCD still had its tracking tracking labels attached to it.

**The Fix**:
Triggered the Ansible playbook to recreate the secret locally, deliberately bypassing Git. Because Ansible injected the YAML without ArgoCD's `app.kubernetes.io/instance` tracking labels, the secret is now fully untethered from the GitOps lifecycle.

---

## 4. Redis Connection Refusal: WRONGPASS
**Symptom**:
After the secret was restored, the fetcher immediately crashed emitting: `could not connect to redis: WRONGPASS invalid username-password pair`.

**How to Debug**:
Compare the credentials requested by the crashed pod against the ones natively running inside the Redis database.
```bash
# 1. Check the crashed fetcher logs
kubectl logs -l app=engine-fetcher

# 2. Extract the actual password the LIVE Redis pod initialized with
kubectl exec -it deployment/redis -- env | grep REDIS_PASSWORD
```
* **Error/Mismatch**: `could not connect to redis: WRONGPASS` and the local password file is fundamentally different from the `exec` environment variable.
* **Expected Output**: `Connected to Redis successfully` in the Fetcher logs.

**Cause**:
The local `ansible/redis-secret.yaml` template used `password:` instead of `REDIS_PASSWORD:`. Furthermore, the user accidentally copy-pasted their encoded ArgoCD password into the YAML variable instead of the database hash.

**The Fix**:
Dynamically patched the `default` namespace secret using `kubectl patch` to include the correct Base64 hash mapped to `REDIS_PASSWORD`. Updated the `.example` template to prevent future engineers from using the wrong array key.

---

## 5. ArgoCD / Helm Security Context API Conflict
**Symptom**:
ArgoCD perpetually marks the `loki-observability` Application as `OutOfSync` and reports a Kubernetes API comparison error regarding privilege escalations.

**How to Debug**:
Check the exact ArgoCD synchronization conditions and patch events.
```bash
# 1. Describe the ArgoCD application to view the Sync failure message
kubectl describe app loki-observability -n argocd | tail -n 20
```
* **Error Output**: `Sync operation to 2.10.2 failed ... Invalid value ... cannot set allowPrivilegeEscalation to false and privileged to true`
* **Expected Output**: `Sync Status: Synced` and `Health Status: Healthy`.

**Cause**:
A manual `kubectl patch` added `privileged: true` to the DaemonSet. During the next sync, ArgoCD attempted to merge the `grafana/promtail` Helm chart config (which hardcodes `allowPrivilegeEscalation: false`) with the live cluster (`privileged: true`). Kubernetes API instantly rejected the illegal combination.

**The Fix**:
Bypassed the Helm container boundaries completely by removing the `containerSecurityContext` overrides and replacing them with a native `seLinuxOptions: type: spc_t` placed directly on the Pod level. The stuck DaemonSet was deleted to allow ArgoCD a clean deployment slate.
