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

export interface IPAddress {
  "ip-address-type": string;
  "ip-address": string;
  prefix: number;
}

export interface NetworkInterface {
  name: string;
  "hardware-address": string;
  "ip-addresses": IPAddress[];
}

export interface BackupInfo {
  volid: string;
  size: number;
  ctime: number;
  notes?: string;
  vmid: number;
  format: string;
  content: string;
}

export interface BackupSchedule {
  id: string;
  type?: string;
  vmid?: string;
  storage: string;
  schedule: string;
  enabled: number;
  comment?: string;
  mode?: string;
  compress?: string;
  node?: string;
}

export interface ContainerConfig {
  hostname?: string;
  ostype?: string;
  cores?: number;
  memory?: number;
  swap?: number;
  rootfs?: string;
  net0?: string;
  searchdomain?: string;
  nameserver?: string;
  unprivileged?: number;
  [key: string]: unknown;
}

export interface ContainerNetInterface {
  name: string;
  hwaddr?: string;
  inet?: string;
  inet6?: string;
  [key: string]: unknown;
}

export interface FilesystemInfo {
  name: string;
  mountpoint: string;
  type: string;
  "total-bytes"?: number;
  "used-bytes"?: number;
}

// Job types for async operations via NATS
export type JobStatus = "pending" | "running" | "completed" | "failed";
export type JobStep = "" | "cloning" | "configuring" | "resizing" | "starting" | "waiting_for_running" | "ready"
  | "backing_up" | "stopping" | "deleting" | "restoring";
export type JobType = "vm" | "container" | "backup_vm" | "backup_container" | "restore_vm" | "restore_container";

export interface Job {
  id: string;
  type: JobType;
  status: JobStatus;
  step: JobStep;
  progress: number;
  error?: string;
  created_at: string;
  updated_at: string;
  source_node: string;
  source_vmid: number;
  target_node: string;
  new_vmid: number;
  name: string;
  ciuser?: string;
  disk_size?: number;
  full_clone: boolean;
  ip_address?: string;
}

export interface JobEvent {
  job_id: string;
  status: JobStatus;
  step: JobStep;
  progress: number;
  error?: string;
  ip_address?: string;
}
