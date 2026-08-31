import type { ActionGate, ProposalGates } from "../../api/types";

export function approvalGate(gates: ProposalGates): ActionGate {
  return gates.approvalAction;
}

export function releaseGate(gates: ProposalGates): ActionGate {
  return gates.releaseAction;
}
