import { fireEvent, render, screen } from "@testing-library/preact";
import { useState } from "preact/hooks";
import { describe, expect, it, vi } from "vitest";
import { PinInput } from "./PinInput";

function PinHarness({
  disabled = false,
  error,
  initialValue = "",
  onChange
}: {
  disabled?: boolean;
  error?: string;
  initialValue?: string;
  onChange?: (value: string) => void;
}) {
  const [pin, setPin] = useState(initialValue);

  return (
    <PinInput
      disabled={disabled}
      error={error}
      id="test-pin"
      label="Cashier PIN"
      onChange={(value) => {
        setPin(value);
        onChange?.(value);
      }}
      value={pin}
    />
  );
}

describe("PinInput", () => {
  it("accepts only digits and stores at most 6 characters", () => {
    const onChange = vi.fn();
    render(<PinHarness onChange={onChange} />);

    fireEvent.input(screen.getByLabelText("Cashier PIN"), {
      target: { value: "12a34b5678" }
    });

    expect(screen.getByLabelText("Cashier PIN")).toHaveValue("123456");
    expect(onChange).toHaveBeenLastCalledWith("123456");
  });

  it("keeps the first 6 digits when a formatted value is pasted", () => {
    render(<PinHarness />);

    fireEvent.input(screen.getByLabelText("Cashier PIN"), {
      target: { value: " 12-34 56 78 " }
    });

    expect(screen.getByLabelText("Cashier PIN")).toHaveValue("123456");
  });

  it("renders six masked visual boxes without exposing clear digits", () => {
    render(<PinHarness initialValue="1234" />);

    const boxes = screen.getAllByTestId("pin-box");

    expect(boxes).toHaveLength(6);
    expect(boxes.slice(0, 4).map((box) => box.textContent)).toEqual(["•", "•", "•", "•"]);
    expect(boxes.slice(4).map((box) => box.textContent)).toEqual(["", ""]);
    expect(screen.queryByText("1234")).not.toBeInTheDocument();
  });

  it("updates when the parent clears the value after a failed login", () => {
    function ClearablePin() {
      const [pin, setPin] = useState("123456");

      return (
        <>
          <PinInput id="clearable-pin" label="Cashier PIN" onChange={setPin} value={pin} />
          <button type="button" onClick={() => setPin("")}>
            Clear
          </button>
        </>
      );
    }

    render(<ClearablePin />);

    fireEvent.click(screen.getByRole("button", { name: "Clear" }));

    expect(screen.getByLabelText("Cashier PIN")).toHaveValue("");
    expect(screen.getAllByTestId("pin-box").every((box) => box.textContent === "")).toBe(true);
  });

  it("does not accept input when disabled", () => {
    const onChange = vi.fn();
    render(<PinHarness disabled onChange={onChange} />);

    expect(screen.getByLabelText("Cashier PIN")).toBeDisabled();
    expect(onChange).not.toHaveBeenCalled();
  });

  it("associates error text with the accessible input", () => {
    render(<PinHarness error="Invalid PIN. Try again." />);

    const input = screen.getByLabelText("Cashier PIN");
    const error = screen.getByText("Invalid PIN. Try again.");

    expect(input).toHaveAttribute("aria-invalid", "true");
    expect(input).toHaveAccessibleDescription("Invalid PIN. Try again.");
    expect(error).toHaveAttribute("id", "test-pin-error");
  });
});
