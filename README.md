# 6.824 Final Project: Distributed Private Electronic Voting

## Team members: 
Carolina Ortega and Felipe Monsalve (cortegap, fmvfelipemonsalve)


## Introduction
Electronic voting is a relevant concept in scenarios like the ongoing pandemic, where in-person interaction is limited, and in cases where you want to expand access to voting for remote or underserved populations.  However, electronic voting presents challenges that must be reliably solved before being an acceptable alternative. Some of these challenges are guaranteeing the privacy of the votes, transparency of the vote counts, as well as guaranteeing that an individual votes at most once.

In this project we tackle the issue of voter privacy, implementing an electronic voting scheme based on the Shamir Secret Sharing algorithm. This voting system uses a counting committee, instead of a central authority, to guarantee vote anonymity assuming honest behavior from the voters and the committee members. We intend to write a basic implementation of the [algorithm](https://inst.eecs.berkeley.edu/~cs276/fa20/psets/pset6.pdf) focusing on its applications for voting, and will explore potential issues that can come up based on the distributed nature of such a system. Some of these issues are network partitions and server crashes, which are especially important to handle since they can affect the correctness of our system.


## Implementation
Our implementation of an electronic voting scheme based on the Shamir Secret Sharing algorithm will have two components: the voters and the counting committee. The voters, which we assume to be set, determine and split their vote into parts using the Shamir Secret Sharing algorithm. Afterwards, they send one part to each counting committee member. Each member of the counting committee would receive parts from all the voters, add them, and share their count with the other members of the committee, to get the total count of votes. Once the total count is ready, the committee will share the winner of the election with the voters.

As we described before, our algorithm assumes well behaved voters and committee members, though we might explore authentication of these actors as a stretch goal. We will be focusing instead on making our system resilient to failures, making sure not only that our system computes the correct outcome, but that it can keep making progress in the face of a reasonable number of fail-stop crashes or network partitions.



## Milestones
We intend to start with a basic implementation of a simple voting system, with a set number of voters and committee members, which assumes no failures. We will then gradually add complexity, expanding our fault tolerance model as described below:
- We will begin implementing a simple voting scheme, where a set threshold of voters must vote for a party, and the committee members must all be available to agree on the winning party. This is a straightforward implementation of the Shamir Secret Sharing algorithm.
- We can implement a simple fix to basic failures by adding a voter timeout, by which voters resubmit their vote if they haven’t heard back from the voting committee about a twinning party. The voters will need UIDs, which the committee members have available, to prevent duplicate votes from being counted.
- To ensure the correctness of our voting system in the face of network failures/ partitions, we must ensure that all committee members have votes from the same members. To ensure this, we will have the committee members compute the largest subset of voters they have in common before exchanging their partial sums to compute a winner.
- To handle voter failures, we will make sure each voter persists its vote before sharing it, so that if it fails and recovers in the same round, it won’t corrupt our voting system by sending different votes to different committee members.
- To be resilient against committee member failures or partitions, we can exploit a feature of the  Shamir Secret Sharing algorithm, which allows us to determine a winner of the vote with a set threshold of committee members.
- (Stretch) There might exist a subset of k committee members that have a larger subset of voters in common than all n committee members. We will optimize the subset of k committee members to maximize the number of voters included in the vote.
- (Stretch) Upon committee member failures, we need to make sure that servers that fail recover (or are replaced), so that committee members don’t decrease to a point where our system can no longer compute a winner. Committee members must communicate within themselves, identify failures (potentially with a majority consensus), and restart committee members.
- (Mega-stretch) We want to explore ways in which we can dynamically determine the voters, so that the voting committee doesn’t necessarily need to know all the voters before the vote. We are unsure if this has any practical uses or straightforward implementations that give us benefits, and we expect to have trouble with network partitions, among other failures.


## Testing
We will use testing tools used for previous labs to test vote integrity in the face of different voter/ committee member crashes, and network partitions. We will build this testing suite gradually to test the capabilities expected at each milestone.
