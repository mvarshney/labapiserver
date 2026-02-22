## k3s Installation and Setup

### 1. Install k3s

```bash
# Install k3s (single node setup)
curl -sfL https://get.k3s.io | sh -

# Wait for k3s to be ready
sudo systemctl status k3s
```

### 2. Set up kubectl access

```bash
# Create .kube directory
mkdir -p ~/.kube

# Copy k3s kubeconfig
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config

# Fix permissions
sudo chown $USER:$USER ~/.kube/config
chmod 600 ~/.kube/config

# Verify connection
kubectl get nodes
```

### 3. Verify k3s is running

```bash
# Check node status
kubectl get nodes

# Check system pods
kubectl get pods -A

# Should see pods like: coredns, traefik, metrics-server, local-path-provisioner
```

### 4. Install ArgoCD

```bash
# Create namespace
kubectl create namespace argocd
```

```bash
# Download the manifest
curl -o argocd-install.yaml https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Apply with server-side apply
kubectl apply -f argocd-install.yaml --server-side --force-conflicts -n argocd

# Wait for ArgoCD to be ready
kubectl wait --for=condition=available --timeout=300s deployment/argocd-server -n argocd

# Get initial admin password
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d && echo
```

### 5. Deploy the App via ArgoCD

```bash
# Apply Kong application
kubectl apply -f iac/argocd/root-app.yaml

# Watch deployment
kubectl get applications -n argocd -w
```

### 6. Access Kong Gateway

```bash
# Wait for Kong to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=kong -n kong --timeout=300s

# Get Kong services
kubectl get svc -n kong

# Get LoadBalancer IP (k3s uses node IP)
kubectl get svc -n kong kong-kong-proxy -o wide

# Test Kong proxy (use your node's IP, usually your machine's IP)
curl http://localhost
```
sd
### 7. Access Kong Admin API

```bash
# Port forward to access admin API
kubectl port-forward -n kong svc/kong-kong-admin 8001:8001

# In another terminal, test admin API
curl http://localhost:8001
```

### 8. Access ArgoCD UI (Optional)

```bash
# Port forward ArgoCD
kubectl port-forward svc/argocd-server -n argocd 8080:443

# Access at: https://localhost:8080
# Username: admin
# Password: (from step 4)
```

## Troubleshooting

### Check k3s logs
```bash
sudo journalctl -u k3s -f
```

### Restart k3s if needed
```bash
sudo systemctl restart k3s
```

### Check Kong pods
```bash
kubectl get pods -n kong
kubectl logs -n kong -l app.kubernetes.io/name=kong
```

### Check ArgoCD application status
```bash
kubectl get applications -n argocd
kubectl describe application kong -n argocd
```
