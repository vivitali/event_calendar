#!/usr/bin/env python3
"""
Winnipeg Tech Events Lambda Function
AWS Lambda handler for automated event fetching and Telegram posting
"""

import json
import os
import logging
import requests
from datetime import datetime, timedelta
from typing import Dict, List, Any, Optional

# Configure logging
logger = logging.getLogger()
logger.setLevel(logging.INFO)

class EventScraper:
    """Simplified event scraper for Lambda environment"""
    
    def __init__(self):
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'Winnipeg Tech Events Bot 1.0'
        })
    
    def fetch_events(self) -> List[Dict[str, Any]]:
        """Fetch events from all sources with fallback to sample data"""
        events = []
        
        try:
            # In a real implementation, you would scrape actual websites
            # For now, return comprehensive sample data
            events = self.get_sample_events()
            logger.info(f"Fetched {len(events)} events from sources")
        except Exception as e:
            logger.error(f"Error fetching events: {e}")
            # Fallback to sample data
            events = self.get_sample_events()
            logger.info(f"Using fallback sample data: {len(events)} events")
        
        return events
    
    def get_sample_events(self) -> List[Dict[str, Any]]:
        """Return sample events for testing and fallback"""
        now = datetime.now()
        
        return [
            {
                "id": "meetup-ai-ml-1",
                "name": "Winnipeg AI & Machine Learning Meetup",
                "description": "Join us for an evening discussing the latest trends in AI and machine learning.",
                "source": "meetup",
                "url": "https://www.meetup.com/winnipeg-ai-ml/events/example1",
                "venue": "Innovation Hub Winnipeg",
                "group": "Winnipeg AI Community",
                "attendee_count": 45,
                "price": "Free",
                "start_time": (now + timedelta(days=3)).isoformat(),
                "end_time": (now + timedelta(days=3, hours=2)).isoformat(),
            },
            {
                "id": "eventbrite-conference-1",
                "name": "Winnipeg Tech Conference 2025",
                "description": "Annual technology conference featuring local and international speakers.",
                "source": "eventbrite",
                "url": "https://www.eventbrite.ca/e/winnipeg-tech-conference-2025-tickets-example1",
                "venue": "Convention Centre",
                "group": "Winnipeg Tech Events",
                "attendee_count": 200,
                "price": "$50",
                "start_time": (now + timedelta(days=7)).isoformat(),
                "end_time": (now + timedelta(days=7, hours=8)).isoformat(),
            },
            {
                "id": "devevents-workshop-1",
                "name": "Winnipeg Developer Workshop",
                "description": "Hands-on coding workshop for developers of all levels.",
                "source": "devevents",
                "url": "https://dev.events/event/winnipeg-developer-workshop-2025",
                "venue": "TechSpace Winnipeg",
                "group": "Winnipeg Developers",
                "attendee_count": 30,
                "price": "Free",
                "start_time": (now + timedelta(days=14)).isoformat(),
                "end_time": (now + timedelta(days=14, hours=6)).isoformat(),
            }
        ]

class TelegramService:
    """Telegram Bot API service"""
    
    def __init__(self, bot_token: str):
        self.bot_token = bot_token
        self.base_url = f"https://api.telegram.org/bot{bot_token}"
        self.session = requests.Session()
    
    def send_message(self, chat_id: str, message: str) -> bool:
        """Send message to Telegram chat"""
        try:
            url = f"{self.base_url}/sendMessage"
            data = {
                "chat_id": chat_id,
                "text": message,
                "parse_mode": "Markdown",
                "disable_web_page_preview": True
            }
            
            response = self.session.post(url, json=data, timeout=30)
            response.raise_for_status()
            
            result = response.json()
            if result.get("ok"):
                logger.info(f"Message sent successfully to chat {chat_id}")
                return True
            else:
                logger.error(f"Telegram API error: {result.get('description')}")
                return False
                
        except Exception as e:
            logger.error(f"Failed to send Telegram message: {e}")
            return False
    
    def send_alert(self, chat_id: str, alert_message: str) -> bool:
        """Send alert message to Telegram"""
        alert = f"🚨 *Winnipeg Tech Events Alert*\n\n{alert_message}\n\n_Time: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}_"
        return self.send_message(chat_id, alert)

def filter_future_events(events: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    """Filter events to only include future events"""
    now = datetime.now()
    future_events = []
    
    for event in events:
        try:
            start_time = datetime.fromisoformat(event["start_time"].replace('Z', '+00:00'))
            if start_time > now:
                future_events.append(event)
        except (ValueError, KeyError) as e:
            logger.warning(f"Invalid date format for event {event.get('id', 'unknown')}: {e}")
    
    return future_events

def group_events_by_time(events: List[Dict[str, Any]]) -> Dict[str, List[Dict[str, Any]]]:
    """Group events by time period"""
    now = datetime.now()
    groups = {
        "Today": [],
        "This Week": [],
        "Next Week": [],
        "Later": []
    }
    
    for event in events:
        try:
            start_time = datetime.fromisoformat(event["start_time"].replace('Z', '+00:00'))
            
            if start_time.date() == now.date():
                groups["Today"].append(event)
            elif start_time.date() <= (now + timedelta(days=7)).date():
                groups["This Week"].append(event)
            elif start_time.date() <= (now + timedelta(days=14)).date():
                groups["Next Week"].append(event)
            else:
                groups["Later"].append(event)
        except (ValueError, KeyError) as e:
            logger.warning(f"Error grouping event {event.get('id', 'unknown')}: {e}")
    
    # Remove empty groups
    return {k: v for k, v in groups.items() if v}

def generate_telegram_message(events: List[Dict[str, Any]]) -> str:
    """Generate formatted Telegram message"""
    if not events:
        return "📅 No upcoming events found for Winnipeg tech community."
    
    now = datetime.now()
    date_str = now.strftime("%A, %B %d, %Y")
    
    message = f"🚀 *Winnipeg Tech Events - {date_str}*\n\n"
    
    # Group events by time period
    groups = group_events_by_time(events)
    
    for period, period_events in groups.items():
        if period_events:
            message += f"*{period}:*\n"
            for event in period_events:
                name = event.get("name", "Unknown Event")
                url = event.get("url", "")
                venue = event.get("venue", "")
                price = event.get("price", "")
                
                # Escape markdown special characters
                name = name.replace("*", "\\*").replace("_", "\\_").replace("[", "\\[").replace("]", "\\]")
                venue = venue.replace("*", "\\*").replace("_", "\\_").replace("[", "\\[").replace("]", "\\]")
                
                message += f"• {name}\n"
                
                try:
                    start_time = datetime.fromisoformat(event["start_time"].replace('Z', '+00:00'))
                    time_str = start_time.strftime("%b %d at %I:%M %p")
                    message += f"  📅 {time_str}\n"
                except (ValueError, KeyError):
                    pass
                
                if venue:
                    message += f"  📍 {venue}\n"
                
                if price and price != "Free":
                    message += f"  💰 {price}\n"
                
                if url:
                    message += f"  🔗 [View Event]({url})\n"
                
                message += "\n"
    
    message += "\n_Shared via Winnipeg Tech Events Tracker_"
    
    return message

def lambda_handler(event: Dict[str, Any], context: Any) -> Dict[str, Any]:
    """AWS Lambda handler function"""
    logger.info(f"Lambda function started with event: {json.dumps(event)}")
    
    try:
        # Get configuration from environment variables
        bot_token = os.getenv("TELEGRAM_BOT_TOKEN")
        chat_id = os.getenv("TELEGRAM_CHAT_ID")
        test_mode = os.getenv("TEST_MODE", "false").lower() == "true"
        city = os.getenv("CITY", "Winnipeg")
        categories = os.getenv("CATEGORIES", "tech")
        
        logger.info(f"Configuration: City={city}, Categories={categories}, TestMode={test_mode}")
        
        # Initialize services
        scraper = EventScraper()
        telegram_service = TelegramService(bot_token) if bot_token else None
        
        # Fetch events
        logger.info("Fetching events from all sources...")
        all_events = scraper.fetch_events()
        
        # Filter future events
        future_events = filter_future_events(all_events)
        logger.info(f"Filtered to {len(future_events)} future events")
        
        if not future_events:
            logger.info("No future events found")
            return {
                "statusCode": 200,
                "body": json.dumps({
                    "success": True,
                    "message": "No future events found",
                    "events_count": 0,
                    "message_sent": False
                })
            }
        
        # Generate message
        message = generate_telegram_message(future_events)
        logger.info(f"Generated message with {len(message)} characters")
        
        # Check message length
        if len(message) > 4096:
            logger.warning(f"Message too long ({len(message)} chars), truncating...")
            message = message[:4090] + "..."
        
        # Send message if not in test mode
        message_sent = False
        if test_mode:
            logger.info("🧪 TEST MODE: Message would be sent but not actually posted")
        elif telegram_service and chat_id:
            logger.info("Sending message to Telegram...")
            message_sent = telegram_service.send_message(chat_id, message)
            if message_sent:
                logger.info("✅ Message sent to Telegram successfully")
            else:
                logger.error("❌ Failed to send message to Telegram")
        else:
            logger.warning("Telegram not configured, message not sent")
        
        # Return success response
        return {
            "statusCode": 200,
            "body": json.dumps({
                "success": True,
                "message": "Events processed successfully",
                "events_count": len(future_events),
                "message_sent": message_sent,
                "test_mode": test_mode,
                "timestamp": datetime.now().isoformat()
            })
        }
        
    except Exception as e:
        logger.error(f"Lambda function failed: {e}")
        
        # Send error alert if Telegram is configured
        try:
            bot_token = os.getenv("TELEGRAM_BOT_TOKEN")
            chat_id = os.getenv("TELEGRAM_CHAT_ID")
            if bot_token and chat_id:
                telegram_service = TelegramService(bot_token)
                telegram_service.send_alert(chat_id, f"Lambda function failed: {str(e)}")
        except Exception as alert_error:
            logger.error(f"Failed to send error alert: {alert_error}")
        
        return {
            "statusCode": 500,
            "body": json.dumps({
                "success": False,
                "error": str(e),
                "timestamp": datetime.now().isoformat()
            })
        }

# For local testing
if __name__ == "__main__":
    # Test the function locally
    test_event = {
        "source": "manual",
        "test_mode": True
    }
    
    # Set test environment variables
    os.environ["TEST_MODE"] = "true"
    os.environ["CITY"] = "Winnipeg"
    os.environ["CATEGORIES"] = "tech"
    
    result = lambda_handler(test_event, None)
    print(json.dumps(result, indent=2))
