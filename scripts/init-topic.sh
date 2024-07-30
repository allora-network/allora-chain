# Define prefix variable
PREFIX="TOPIC SCRIPT:"

# echo all commands
set -x

# # addresses
worker1="allo1tvh6nv02vq6m4mevsa9wkscw53yxvfn7xt8rud"
worker2="allo12vncm038gpyr2u2v524pgqmmdg39uqn3qgnjjc"
reputer1="allo15k879zq9thpr6guqpc3w4lcu6lp62v5pf9yl4p"
reputer2="allo1rf9tk6f05aaek086w6qvwjpjlwy2s8kvtkagrc"

# Funding accounts
allorad tx bank send alice $worker1   100000000uallo  --yes
sleep 5
allorad tx bank send alice $worker2   100000000uallo --yes
sleep 5
allorad tx bank send alice $reputer1  100000000uallo --yes
sleep 5
allorad tx bank send alice $reputer2  100000000uallo --yes
sleep 5

#### Create accounts and topics
alice_addr=$(allorad keys show alice -a)
echo "$PREFIX Alice address: $alice_addr"

# "create-topic [creator] [metadata] [loss_method] [epoch_length] [ground_truth_lag] [worker_submission_window] [p_norm] [alpha_regret] [allow_negative] [epsilon]",
# create a topic
echo "$PREFIX Creating topic..."
NEXT_TOPIC_ID=$(allorad query emissions next-topic-id | grep -oE 'next_topic_id: "[0-9]+' | grep -oE '[0-9]+')
while [ -z "$NEXT_TOPIC_ID" ] || [ "$NEXT_TOPIC_ID" -lt 2 ]; do
    allorad tx emissions create-topic $alice_addr "ETH 24h Prediction" mse  \
    60 60 50 "3" "1" false "0.01" --yes
    sleep 5
    NEXT_TOPIC_ID=$(allorad query emissions next-topic-id | grep -oE 'next_topic_id: "[0-9]+' | grep -oE '[0-9]+')
done

# if topic is not created then exit
if [ -z "$NEXT_TOPIC_ID" ] || [ "$NEXT_TOPIC_ID" -lt 2 ]; then
    echo "$PREFIX Topic not created. Exiting..."
    exit 1
fi
echo "$PREFIX Topic created with id: $NEXT_TOPIC_ID"

# request inference for topic
echo "$PREFIX Funding topic..."
RETRY_COUNT=0
MAX_RETRY=3
FUND_AMOUNT=100000000
while true; do
    echo "$PREFIX Funding topic..."
    allorad  tx emissions  fund-topic  $alice_addr 1 $FUND_AMOUNT  --yes
    if [ $RETRY_COUNT -eq $MAX_RETRY ]; then
        echo "$PREFIX End funding topic."
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT+1))
    sleep 5
done


echo "$PREFIX ** REPUTER **"
# Create and fund reputer
allorad keys add myreputer
reputer_addr=$(allorad keys show myreputer -a)
sleep 5
echo "$PREFIX Sending funds to reputer..."
allorad tx bank send alice $reputer_addr  100000000uallo --yes
sleep 5

# Register reputer
echo "$PREFIX Registering reputer..."
allorad tx emissions register $reputer_addr 1 $reputer_addr "True" --yes
sleep 5

# Stake reputer on topic 1
echo "$PREFIX Adding stake to reputer..."
allorad tx emissions add-stake $reputer_addr  1  90000000 --yes

sleep 10
# Check if topic is active
echo "$PREFIX Checking if topic is active..."
# Assuming a command to check topic status (this command is hypothetical)
# allorad query emissions active-topics '{"limit":10}' | grep -q "topic_id: 1"
if ! allorad query emissions active-topics '{"limit":10}' | grep -q "id: \"1"; then
  echo "No active topic with topic_id: 1 found. Exiting..."
  exit 1
fi

echo "$PREFIX ** INFERER **"
# Create and fund worker
allorad keys add myworker1
worker_addr=$(allorad keys show myworker1 -a)
sleep 5
echo "$PREFIX Sending funds to worker..."
allorad tx bank send alice $worker_addr  100000000uallo --yes
sleep 5

# Register worker
echo "$PREFIX Registering worker..."
allorad tx emissions register $worker_addr 1 $worker_addr "False" --yes
sleep 5


echo "$PREFIX ** END INIT SCRIPT **"