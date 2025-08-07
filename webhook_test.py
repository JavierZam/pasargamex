#!/usr/bin/env python3
import json
import requests
import sys

def test_webhook(order_id, transaction_status="settlement"):
    """Test the Midtrans webhook endpoint"""
    
    # Sample webhook payload from Midtrans
    webhook_payload = {
        "order_id": order_id,
        "transaction_status": transaction_status,
        "fraud_status": "accept",
        "status_code": "200",
        "gross_amount": "15000.00",
        "payment_type": "credit_card",
        "transaction_time": "2025-08-06 22:37:00",
        "signature_key": "dummy_signature_for_sandbox"
    }
    
    print(f"Testing webhook for order: {order_id}")
    print(f"Transaction status: {transaction_status}")
    print(f"Payload: {json.dumps(webhook_payload, indent=2)}")
    
    try:
        # Send webhook to local server
        response = requests.post(
            "http://localhost:8080/v1/payments/midtrans/callback",
            json=webhook_payload,
            headers={
                "Content-Type": "application/json",
                "X-Midtrans-Signature": "dummy_signature_for_sandbox"
            },
            timeout=10
        )
        
        print(f"\nResponse Status: {response.status_code}")
        print(f"Response Body: {response.text}")
        
        if response.status_code == 200:
            print("✅ Webhook processed successfully!")
        else:
            print("❌ Webhook failed to process")
            
    except requests.exceptions.RequestException as e:
        print(f"❌ Error sending webhook: {e}")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 webhook_test.py <order_id> [transaction_status]")
        print("Example: python3 webhook_test.py PGX-abc123-1691234567 settlement")
        sys.exit(1)
    
    order_id = sys.argv[1]
    transaction_status = sys.argv[2] if len(sys.argv) > 2 else "settlement"
    
    test_webhook(order_id, transaction_status)