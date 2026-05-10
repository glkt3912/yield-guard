import base64
import json
import os

import functions_framework
import google.auth
from googleapiclient import discovery


@functions_framework.cloud_event
def stop_cloud_run(cloud_event):
    data = base64.b64decode(cloud_event.data["message"]["data"]).decode("utf-8")
    payload = json.loads(data)

    cost_amount = float(payload.get("costAmount", 0))
    budget_amount = float(payload.get("budgetAmount", 1))

    if cost_amount < budget_amount:
        print(f"Cost {cost_amount} < budget {budget_amount}, no action needed")
        return

    project_id = os.environ["PROJECT_ID"]
    region = os.environ["REGION"]
    service_name = os.environ["SERVICE_NAME"]

    credentials, _ = google.auth.default()

    service = discovery.build("run", "v2", credentials=credentials)
    full_name = f"projects/{project_id}/locations/{region}/services/{service_name}"

    try:
        service.projects().locations().services().patch(
            name=full_name,
            updateMask="scaling.maxInstanceCount",
            body={"scaling": {"maxInstanceCount": 0}},
        ).execute()
        print(f"Cloud Run service {service_name} stopped (max_instance_count=0)")
    except Exception as e:
        print(f"Failed to stop Cloud Run service {service_name}: {e}")
        raise
