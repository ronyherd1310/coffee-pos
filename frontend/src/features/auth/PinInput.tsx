import type { Ref } from "preact";
import type { JSX } from "preact/jsx-runtime";

type PinInputProps = {
  disabled?: boolean;
  describedById?: string;
  error?: string;
  helperText?: string;
  id: string;
  inputRef?: Ref<HTMLInputElement>;
  invalid?: boolean;
  label: string;
  onChange: (value: string) => void;
  onKeyDown?: JSX.KeyboardEventHandler<HTMLInputElement>;
  value: string;
};

const PIN_LENGTH = 6;

export function PinInput({
  disabled = false,
  describedById,
  error,
  helperText,
  id,
  inputRef,
  invalid = false,
  label,
  onChange,
  onKeyDown,
  value
}: PinInputProps) {
  const errorId = error ? `${id}-error` : undefined;
  const descriptionId = errorId ?? describedById;
  const isInvalid = Boolean(error || invalid);

  return (
    <div className="pin-field">
      <label className="pin-label" htmlFor={id}>
        {label}
      </label>
      {helperText ? <p className="login-helper">{helperText}</p> : null}
      <div className="pin-control">
        <input
          aria-describedby={descriptionId}
          aria-invalid={isInvalid ? "true" : "false"}
          autoComplete="current-password"
          className="pin-control__input"
          disabled={disabled}
          id={id}
          inputMode="numeric"
          ref={inputRef}
          maxLength={PIN_LENGTH}
          onInput={(event) => onChange(normalizePin(event.currentTarget.value))}
          onKeyDown={onKeyDown}
          pattern="[0-9]*"
          type="password"
          value={value}
        />
        <div className="pin-control__boxes" aria-hidden="true">
          {Array.from({ length: PIN_LENGTH }).map((_, index) => (
            <span className="pin-box" data-testid="pin-box" key={index}>
              {index < value.length ? "•" : ""}
            </span>
          ))}
        </div>
      </div>
      {error ? (
        <p className="pin-error" id={errorId}>
          {error}
        </p>
      ) : null}
    </div>
  );
}

function normalizePin(value: string) {
  return value.replace(/\D/g, "").slice(0, PIN_LENGTH);
}
