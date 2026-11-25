import { Activity, CircleAlert, CircleCheck } from "lucide-react";

import { useHealthQuery } from "../queries/health";

export function ServiceStatus(): React.JSX.Element {
  const health = useHealthQuery();

  if (health.isPending) {
    return (
      <div className="flex items-center gap-3 text-content-secondary">
        <Activity aria-hidden="true" className="size-5 animate-pulse" />
        <span>正在连接</span>
      </div>
    );
  }

  if (health.isError) {
    return (
      <div className="flex items-center gap-3 text-state-danger">
        <CircleAlert aria-hidden="true" className="size-5" />
        <span>服务不可用</span>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-3 text-state-success">
      <CircleCheck aria-hidden="true" className="size-5" />
      <span>{health.data.service} 已连接</span>
    </div>
  );
}
