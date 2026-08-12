export type Ledger = {
  id: string;
  name: string;
  description: string;
  createdAt: string;
};

export type Change = {
  id: string;
  ledgerId: string;
  sequence: number;
  unit: string;
  action: "PUT" | "DELETE";
  desired: unknown;
  baseFingerprint: string;
  desiredFingerprint: string;
  status: string;
  createdAt: string;
};

export type Proposal = {
  id: string;
  ledgerId: string;
  title: string;
  baseReleaseId?: string;
  hash: string;
  status: string;
  changeIds: string[];
  createdAt: string;
};

export type Release = {
  id: string;
  ledgerId: string;
  proposalId: string;
  parentId?: string;
  hash: string;
  createdAt: string;
};

export type SystemStatus = {
  status: string;
  version: string;
  inference: string;
};

export type APIError = { error: { code: string; message: string } };
