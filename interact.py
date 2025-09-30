import os
from allora import AlloraAPIClient, ChainSlug

client = AlloraAPIClient(chain_slug=ChainSlug.TESTNET, api_key=os.environ.get("ALLORA_API_KEY"))
