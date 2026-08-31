import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AlertIcon, ChangeIcon, CheckIcon, ChevronIcon, CopyIcon, LedgerIcon, PlusIcon, ProposalIcon, ReleaseIcon, SpinnerIcon } from "./icon";

describe("icons", () => {
  it("renders every decorative icon hidden from the accessibility tree", () => {
    const { container } = render(<><LedgerIcon /><ChangeIcon /><ProposalIcon /><ReleaseIcon /><CheckIcon /><AlertIcon /><CopyIcon /><ChevronIcon /><PlusIcon /><SpinnerIcon /></>);
    const icons = container.querySelectorAll("svg");
    expect(icons).toHaveLength(10);
    icons.forEach((icon) => expect(icon).toHaveAttribute("aria-hidden", "true"));
  });
});
