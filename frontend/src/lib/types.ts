export interface ClusterResource {
  id: string;
  type: "node" | "qemu" | "lxc" | "storage";
  node: string;
  vmid?: number;
  name?: string;
  status: string;
  cpu?: number;
  maxcpu?: number;
  mem?: number;
  maxmem?: number;
  disk?: number;
  maxdisk?: number;
  uptime?: number;
  template?: number;
  pool?: string;
  storage?: string;
  content?: string;
  plugintype?: string;
  ip?: string;
}

export interface ClusterSummary {
  nodes: number;
  nodes_online: number;
  vms_running: number;
  vms_total: number;
  containers_running: number;
  containers_total: number;
  cpu_usage: number;
  cpu_total: number;
  mem_used: number;
  mem_total: number;
  disk_used: number;
  disk_total: number;
}

export interface VMStatus {
  vmid: number;
  name: string;
  status: string;
  cpu: number;
  cpus: number;
  mem: number;
  maxmem: number;
  disk: number;
  maxdisk: number;
  netin: number;
  netout: number;
  diskread: number;
  diskwrite: number;
  uptime: number;
  pid?: number;
  template?: number;
}

export interface ContainerStatus {
  vmid: number;
  name: string;
  status: string;
  type: string;
  cpu: number;
  cpus: number;
  mem: number;
  maxmem: number;
  disk: number;
  maxdisk: number;
  netin: number;
  netout: number;
  uptime: number;
  template?: number;
}

export interface StorageInfo {
  storage: string;
  type: string;
  content: string;
  total: number;
  used: number;
  avail: number;
  active: number;
  enabled: number;
  shared: number;
}

export interface TemplateInfo extends ClusterResource {
  vmtype: "qemu" | "lxc";
}

export interface CloneRequest {
  newid: number;
  name: string;
  target?: string;
  full?: boolean;
}

export interface CreateVMRequest {
  vmid: number;
  name: string;
  node: string;
  ostype: string;
  cores: number;
  memory: number;
  diskSize: number;
  storage: string;
  iso?: string;
}

export interface DeleteRequest {
  node: string;
  vmid: number;
  type: "qemu" | "lxc";
}
