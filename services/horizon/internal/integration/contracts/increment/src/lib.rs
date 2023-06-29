#![no_std]
use soroban_sdk::{contractimpl, log, Env, Symbol};

const COUNTER: Symbol = Symbol::short("COUNTER");

pub struct IncrementContract;

#[contractimpl]
impl IncrementContract {
    /// Increment increments an internal counter, and returns the value.
    pub fn increment(env: Env) -> u32 {
        let mut count: u32 = 0;

        // Get the current count.
        if env.storage().has(&COUNTER) {
            count = env
                .storage()
                .get(&COUNTER)
                .unwrap(); // Panic if the value of COUNTER is not u32.
        }
        log!(&env, "count: {}", count);


        // Increment the count.
        count += 1;

        // Save the count.
        env.storage().set(&COUNTER, &count);

        // Return the count to the caller.
        count
    }
}

mod test;
