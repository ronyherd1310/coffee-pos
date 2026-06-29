import { useEffect, useState } from "preact/hooks";
import { fetchHealth, type HealthResponse } from "./lib/health";

type HealthState =
  | { status: "loading" }
  | { status: "ready"; data: HealthResponse }
  | { status: "error"; message: string };

export function App() {
  const [health, setHealth] = useState<HealthState>({ status: "loading" });

  useEffect(() => {
    let isCurrent = true;

    fetchHealth()
      .then((data) => {
        if (isCurrent) {
          setHealth({ status: "ready", data });
        }
      })
      .catch((error: unknown) => {
        if (isCurrent) {
          setHealth({
            status: "error",
            message: error instanceof Error ? error.message : "Backend health check failed"
          });
        }
      });

    return () => {
      isCurrent = false;
    };
  }, []);

  return (
    <main className="app-shell">
      <section className="intro" aria-labelledby="page-title">
        <p className="eyebrow">Foundation scaffold</p>
        <h1 id="page-title">Coffee POS</h1>
        <p className="summary">
          A small coffee shop point-of-sale foundation is running. Application workflows will be added
          in later slices.
        </p>
      </section>

      <section className="status-panel" aria-labelledby="health-title">
        <div>
          <p className="panel-label">Backend</p>
          <h2 id="health-title">Health status</h2>
        </div>
        <HealthStatus health={health} />
      </section>
    </main>
  );
}

function HealthStatus({ health }: { health: HealthState }) {
  if (health.status === "loading") {
    return (
      <p className="status-message" role="status">
        Checking backend health...
      </p>
    );
  }

  if (health.status === "error") {
    return (
      <p className="status-message status-message--error" role="alert">
        Backend unavailable: {health.message}
      </p>
    );
  }

  return (
    <dl className="health-list" aria-label="Backend health details">
      <div>
        <dt>Status</dt>
        <dd>
          <span className="status-dot" aria-hidden="true" />
          {health.data.status}
        </dd>
      </div>
      <div>
        <dt>Service</dt>
        <dd>{health.data.service}</dd>
      </div>
    </dl>
  );
}
