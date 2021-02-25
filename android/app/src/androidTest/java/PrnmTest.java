// Copyright (c) 2021 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of
// perun-eth-mobile. Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package network.perun.app;

import prnm.*;

import java.math.*;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicReference;

import android.util.Log;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.jupiter.api.Assertions;
import androidx.test.runner.AndroidJUnit4;
import androidx.test.core.app.ApplicationProvider;
import static com.google.common.truth.Truth.assertThat;
import static org.awaitility.Awaitility.*;
import static org.hamcrest.Matchers.is;
import static org.hamcrest.Matchers.notNullValue;

@RunWith(AndroidJUnit4.class)
public class PrnmTest implements prnm.NewChannelCallback, prnm.ProposalHandler, prnm.UpdateHandler, prnm.ConcludedEventHandler {

    /**
     * This class contains four entry points for the test-runner; two per peer and test case.
     *
     * runFirst* sets up a channel, sends TXs and persists it.
     * runSecond* asserts that the channel can be restored from persistence.
     */

    Client client;
    Context ctx = Prnm.contextWithTimeout(120);
    String prefix;
    Setup setup;

    AtomicInteger receivedTx = new AtomicInteger(0);
    AtomicBoolean watcherStopped = new AtomicBoolean(false);
    // handleConcluded returned
    AtomicBoolean concluded = new AtomicBoolean(false);
    AtomicReference<PaymentChannel> ch = new AtomicReference<PaymentChannel>(null);

    @Test
    // Alice entry point for the first test.
    public void runFirstTestAlice() throws Exception {
        runFirstTest(Setup.ALICE);
    }

    @Test
    // Bob entry point for the test runner to run the first test.
    public void runFirstTestBob() throws Exception {
        // Bob will deploy the contracts on his first start.
        runFirstTest(Setup.BOB_NO_CONTRACTS);
    }

    @Test
    // Alice entry point for the test runner to run the second test.
    public void runSecondTestAlice() throws Exception {
        runSecondTest(Setup.ALICE);
    }

    @Test
    // Bob entry point for the test runner to run the second test.
    public void runSecondTestBob() throws Exception {
        runSecondTest(Setup.BOB);
    }

    /**
     * This function does the following:
     *  both:
     *      - sharedSetup
     *      - enables persistence
     *  Alice:
     *      - proposes a payment channel to Bob
     *  Bob:
     *      - accepts the proposed channel from Alice
     *  both (repeated 3 times):
     *      - send 1 TX and wait for a TX from the other peer, Alice goes first
     *      - exit without closing the channel
     * @param s
     * @throws Exception
     */
    public void runFirstTest(Setup s) throws Exception {
        sharedSetup(s);
        if (s.Index == 0) {
            proposeChannel(s);
            sendTx(false, 3);
        }
        else {
            acceptChannel();
            sendTx(true, 5);
        }
        assertOffChainBals(106, 94);
    }

    /**
     * This function does the following:
     *  both:
     *      - sharedSetup
     *      - restores the channel with persistence
     *  both (repeated 3 times):
     *      - send 1 TX and wait for a TX from the other peer, Bob goes first
     *  Alice:
     *      - asserts the final balances
     *  Bob:
     *      - finalizes the channel
     *  both:
     *      - settle the channel
     * @param s
     * @throws Exception
     */
    public void runSecondTest(Setup s) throws Exception {
        sharedSetup(s);
        if (s.Index == 0)
            sendTx(true, 1);
        else
            sendTx(false, 2);
        sharedTeardown(s);

        assertOffChainBals(109, 91);
        assertOnChainBals(1009, 991);
    }

    /**
     * This function:
     * - imports an account from the Setup
     * - connects to the Ethereum node
     * - creates a prnm client
     * - adds the other peer to the client
     * - starts the proposal, update and newChannel handlers
     */
    public void sharedSetup(Setup s) throws Exception {
        this.setup = s;
        Prnm.setLogLevel(4);    // INFO
        prefix = "prnm-" + Setup.Aliases[s.Index];   // android log prefix
        String appDir = ApplicationProvider.getApplicationContext().getFilesDir().getAbsolutePath();
        String ksPath = appDir + "/keystore";
        String dbPath = appDir + "/database";

        Wallet wallet = Prnm.newWallet(ksPath, "0123456789");
        Address onChain = wallet.importAccount(Setup.SKs[s.Index]);

        String ethUrl = "ws://10.5.0.9:8545";
        Config cfg = new Config(Setup.Aliases[s.Index], onChain, s.Adjudicator, s.Assetholder, ethUrl, "0.0.0.0", 5750);
        client = new Client(ctx, cfg, wallet);

        client.addPeer(Setup.Addresses[1-s.Index], Setup.Hosts[1-s.Index], Setup.Ports[1-s.Index]);
        client.onNewChannel(this);
        new Thread(() -> {
            client.handle(this, this);
        }).start();

        client.enablePersistence(dbPath);
        client.restore(ctx);
    }

    public void proposeChannel(Setup s) throws Exception {
        Thread.sleep(5000);
        BigInts initBals = Prnm.newBalances(eth(100), eth(100));
        log("Opening Channel…");
        client.proposeChannel(ctx, Setup.Addresses[1-s.Index], 60, initBals);
        Thread.sleep(100);  // BUG in go-perun. Bob otherwise reports: 'received update for unknown channel'
        log("Channel opened.");
    }

    public void acceptChannel() {
        log("Waiting for channel…");
        await().atMost(20, TimeUnit.SECONDS).untilAtomic(ch, notNullValue());
        log("Channel opened.");
    }

    /**
     * @param wait Decides whether this peer waits first or sends an update first.
     */
    public void sendTx(boolean wait, int amount) throws Exception {
        for (int i = 0; i < 3; ++i) {
            if (wait)
                await().atMost(20, TimeUnit.SECONDS).untilAtomic(receivedTx, is(i +1));
            log("Sending TX…");
            ch.get().send(ctx, eth(amount));
            log("TX sent.");
            if (!wait)
                await().atMost(20, TimeUnit.SECONDS).untilAtomic(receivedTx, is(i +1));
        }
    }

    public void sharedTeardown(Setup s) throws Exception {
        if (s.Index == 1) {
            Thread.sleep(1000);
            log("Finalizing");
            ch.get().finalize(ctx);
        } else {
            Thread.sleep(5000);
            log("Settling");
            ch.get().settle(ctx, false);
        }

        await().atMost(20, TimeUnit.SECONDS).untilAtomic(concluded, is(true));
        ch.get().close();
        await().atMost(20, TimeUnit.SECONDS).untilAtomic(watcherStopped, is(true));
    }

    @Override
    public void onNew(PaymentChannel channel) {
        log("onNewChannel");
        ch.set(channel);
        new Thread(() -> {
            Assertions.assertDoesNotThrow(() -> {
                log("Watcher started");
                channel.watch(this);
                log("Watcher stopped");
                watcherStopped.set(true);
            });
        }).start();
        log("onNewChannel done");
    }

    @Override
    public void handleProposal(ChannelProposal proposal, ProposalResponder responder) {
        Assertions.assertDoesNotThrow(() -> {
            BigInts bals = proposal.getInitBals();
            log("handleProposal Bals: " + bals.get(0).toString() + "/" + bals.get(1).toString() + " CD: " + proposal.getChallengeDuration() + "s");
            responder.accept(ctx);
            log("handleProposal done");
        });
    }

    @Override
    public void handleUpdate(ChannelUpdate update, UpdateResponder responder) {
        Assertions.assertDoesNotThrow(() -> {
            log("handleUpdate");
            responder.accept(ctx);
            log("handleUpdate done #" +receivedTx.incrementAndGet());
        });
    }

    @Override
    public void handleConcluded(byte[] id) {
        if (this.setup.Index == 1) {
            log("handleConcluded");
            Assertions.assertDoesNotThrow(() -> {
                ch.get().settle(ctx, true);
                log("handleConcluded: Settled");
            });
        } else {
            log("handleConcluded: skipped");
        }
        concluded.set(true);
    }

    private void assertOffChainBals(int alice, int bob) throws Exception {
        BigInts bals = ch.get().getState().getBalances();
        log("off-chain Bals: " + bals.get(0).toString() + "/" + bals.get(1).toString());
        assertThat(bals.get(0).cmp(eth(alice))).isEqualTo(0);
        assertThat(bals.get(1).cmp(eth(bob))).isEqualTo(0);
    }

    private void assertOnChainBals(int alice, int bob) throws Exception {
        BigInt aliceBal = client.onChainBalance(ctx, Setup.Addresses[0]);
        BigInt bobBal = client.onChainBalance(ctx, Setup.Addresses[1]);
        log("on-chain Bals: " + aliceBal.toString() + "/" + bobBal.toString());
        // 1/10 ETH
        BigInt deciEth = Prnm.newBigIntFromString("10000000000000000");
        assertThat(aliceBal.isWithin(eth(alice), deciEth)).isTrue();
        assertThat(bobBal.isWithin(eth(bob), deciEth)).isTrue();
    }

    static BigInt eth(int i) {
        return Prnm.newBigIntFromBytes(BigInteger.valueOf(i).multiply(new BigInteger("1000000000000000000")).toByteArray());
    }

    public void log(String msg) {
        Log.i(prefix, msg);
    }
}

class Setup {
    public final static String[] Aliases = {"Alice", "Bob"};
    public final static Address[] Addresses = {new Address("0x05e71027e7d3bd6261de7634cf50F0e2142067C4"), new Address("0xA298Fc05bccff341f340a11FffA30567a00e651f")};
    public final static String[] SKs = {"0x6aeeb7f09e757baa9d3935a042c3d0d46a2eda19e9b676283dce4eaf32e29dc9", "0x7d51a817ee07c3f28581c47a5072142193337fdca4d7911e58c5af2d03895d1a"};
    public final static String[] Hosts = {"10.5.0.6", "10.5.0.6"};
    public final static int[] Ports = {5753, 5750};

    public int Index; // 0 = Alice, 1 = Bob
    public Address Adjudicator, Assetholder;

    private Setup(int index, Address adj, Address asset) {
        Index = index;
        Adjudicator = adj;
        Assetholder = asset;
    }

    public final static Setup ALICE = new Setup(0,
            new Address("0x94503e14e26a433c0802e04f2ac1bb1ce77321f5"), new Address("0xc2f95e626123a61bed88752475b870efc4a5f453"));

    public final static Setup BOB = new Setup(1,
            new Address("0x94503e14e26a433c0802e04f2ac1bb1ce77321f5"), new Address("0xc2f95e626123a61bed88752475b870efc4a5f453"));
    public final static Setup BOB_NO_CONTRACTS = new Setup(1, null, null);
}
