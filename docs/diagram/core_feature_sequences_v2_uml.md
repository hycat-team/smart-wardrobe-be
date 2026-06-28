# Smart Wardrobe - Core Feature Sequence Diagrams v2.1

> Version v2.1 for UML submission.  
> Scope: simplified sequence diagrams based on the latest core feature specification.  
> Format: Mermaid sequence diagrams.

> Participant naming convention:
> - `Member`, `Seller`, `Buyer`, `Viewer`: user-facing actors.
> - `FE`: Frontend Web.
> - `BE`: Backend Server.
> - `DB`: Database.
> - External services keep their service names, such as `AI`, `PayOS`, `Cloudinary`, and `MQ`.

---

## 1. Automated Wardrobe Digitization

```mermaid
sequenceDiagram
    autonumber
    actor Member as Member
    participant FE as Frontend Web
    participant BE as Backend Server
    participant Cloudinary as Cloudinary Service
    participant MQ as Message Queue
    participant WardrobeWorker as Wardrobe Batch Worker
    participant AI as AI Service
    participant DB as Database

    Member->>FE: Select clothing images
    FE->>BE: GET /api/v1/wardrobe-items/upload-signature
    BE-->>FE: Return upload signature
    FE->>Cloudinary: Upload images with signature
    Cloudinary-->>FE: Return ImageUrl and ImagePublicID

    FE->>BE: POST /api/v1/wardrobe-items/batch-upload
    BE->>DB: Check wardrobe quota

    alt Wardrobe limit exceeded
        BE-->>FE: Return ErrWardrobeLimitExceeded
    else Quota is valid
        BE->>DB: Bulk create wardrobe items with status processing
        BE->>MQ: Publish wardrobe.batch_upload events
        BE-->>FE: Return processing items
        FE-->>Member: Show processing status
    end

    MQ->>WardrobeWorker: Deliver wardrobe.batch_upload event
    WardrobeWorker->>AI: AnalyzeFashionImage(ImageUrl)
    AI-->>WardrobeWorker: Return fashion metadata and dominant color
    WardrobeWorker->>WardrobeWorker: Compute HSL and build rich text context
    WardrobeWorker->>AI: GenerateEmbeddings(Rich Text Context)
    AI-->>WardrobeWorker: Return embedding vector

    alt Processing succeeds
        WardrobeWorker->>DB: Update item metadata, HSL, vector, status in_wardrobe
        WardrobeWorker->>MQ: Publish wardrobe.event.created
    else Processing fails after retries
        WardrobeWorker->>DB: Update item status failed
    end
```

---

## 2. AI Outfit Recommendation and Save Outfit

```mermaid
sequenceDiagram
    autonumber
    actor Member as Member
    participant FE as Frontend Web
    participant BE as Backend Server
    participant AI as AI Service
    participant DB as Database

    Member->>FE: Request outfit recommendation
    FE->>BE: POST /api/v1/ai/outfit-recommendations
    BE->>DB: Check daily outfit quota

    alt Outfit quota exceeded
        BE-->>FE: Return ErrOutfitQuotaExceeded
    else Quota is available
        alt Structured input is provided
            BE->>DB: Filter in_wardrobe items by ColorTone, Occasion, Style
        else FreeTextQuery is provided or empty
            BE->>AI: GenerateEmbeddings(FreeTextQuery or casual default)
            AI-->>BE: Return query vector
            BE->>DB: Vector search top 15-20 available items
        end

        alt No suitable candidate items
            BE-->>FE: Return error without quota deduction
        else Candidate items found
            BE->>AI: Generate outfit from candidate items

            alt AI generation succeeds
                AI-->>BE: Return outfit JSON with Primary and Alternatives
            else AI generation fails
                BE->>BE: Build fallback outfit using local HSL matching
            end

            BE->>BE: Validate selected ItemIDs against candidate items
            BE->>DB: Deduct one outfit quota
            BE-->>FE: Return outfit recommendation with IsFallback flag
            FE-->>Member: Display recommended outfit
        end
    end

    opt User saves the outfit
        Member->>FE: Save outfit
        FE->>BE: POST /api/v1/outfits
        BE->>DB: Check MaxOutfits limit
        alt Outfit limit exceeded
            BE-->>FE: Return error
        else Outfit limit is valid
            BE->>DB: Create Outfit and OutfitItems
            BE->>DB: Update selected items last_used_at
            BE-->>FE: Return HTTP 201 Created
        end
    end
```

---

## 3. AI Style Stylist Chatbot

```mermaid
sequenceDiagram
    autonumber
    actor Member as Member
    participant FE as Frontend Web
    participant BE as Backend Server
    participant AI as AI Service
    participant DB as Database

    Member->>FE: Send stylist chat message
    FE->>BE: POST /api/v1/ai/chat/sessions/:contextID/messages/stream
    BE->>DB: Check daily AI chat quota

    alt Chat quota exceeded
        BE-->>FE: Return quota error
    else Quota is available
        BE->>BE: Detect outfit creation intent

        alt Outfit intent detected
            BE-->>FE: Send done event with redirect message
            Note over BE: No AI reasoning and no quota deduction
        else Normal fashion advice message
            BE->>DB: Get recent messages, ContextSummary, wardrobe items
            DB-->>BE: Return conversation and wardrobe context
            BE->>AI: GenerateTextStream(SystemPrompt, UserContent)

            loop Stream response chunks
                AI-->>BE: Return text chunk
                BE-->>FE: Send SSE chunk event
            end

            alt Client disconnects before completion
                BE->>AI: Cancel AI stream
                Note over BE: Do not save response and do not deduct quota
            else Stream completes successfully
                BE-->>FE: Send SSE done event
                BE->>DB: Save user message and AI message
                BE->>DB: Deduct one AI chat quota
            end

            opt Uncompressed messages reach threshold
                BE->>AI: Summarize old messages with previous ContextSummary
                AI-->>BE: Return new ContextSummary
                BE->>DB: Update ContextSummary and remove compressed raw messages
            end
        end
    end
```

---

## 4. P2P Marketplace Item Transfer

```mermaid
sequenceDiagram
    autonumber
    actor Seller as Seller
    actor Buyer as Buyer
    participant BE as Backend Server
    participant DB as Database

    Seller->>BE: POST /api/v1/posts
    BE->>DB: Validate item ownership and active transfer state

    alt Sale post is invalid
        BE-->>Seller: Return validation error
    else Sale post is valid
        BE->>DB: Create Post and PostItems with status available
        BE->>DB: Update seller wardrobe items to selling
        BE->>DB: Sync post total price
        BE-->>Seller: Return HTTP 201 Created
    end

    Buyer->>BE: POST /api/v1/transfers/requests
    BE->>DB: Validate buyer, PostItem status, and TransferState

    alt Transfer request is invalid
        BE-->>Buyer: Return validation error
    else Transfer request is valid
        BE->>DB: Create TransferRequest with status pending
        BE-->>Buyer: Return HTTP 201 Created
    end

    Seller->>BE: POST /api/v1/transfers/mark-sold
    BE->>DB: Validate seller ownership and selected buyer
    BE->>DB: Set selected request accepted and other requests rejected
    BE->>DB: Update PostItems to sold with TransferState pending
    BE->>DB: Hide sibling PostItems from other posts
    BE->>DB: Sync related post total prices
    BE-->>Seller: Return HTTP 200 OK

    alt Buyer confirms receiving items
        Buyer->>BE: POST /api/v1/transfers/accept
        BE->>DB: Validate buyer and pending transfer state
        BE->>DB: Copy items to buyer wardrobe with status in_wardrobe
        BE->>DB: Update seller wardrobe items to sold
        BE->>DB: Update PostItems TransferState to accepted
        BE->>DB: Keep sibling PostItems hidden and clear temporary transfer fields
        BE->>DB: Sync related post total prices
        BE-->>Buyer: Return HTTP 200 OK
    else Buyer declines transfer
        Buyer->>BE: POST /api/v1/transfers/decline
        BE->>DB: Validate buyer and pending transfer state
        BE->>DB: Update PostItems to available with TransferState declined
        BE->>DB: Update buyer TransferRequest to canceled
        BE->>DB: Check remaining active transfers for the same item

        alt No active transfer remains
            BE->>DB: Update seller wardrobe items to selling
            BE->>DB: Restore sibling PostItems to available
        else Another active transfer still exists
            Note over BE,DB: Keep sibling visibility and seller item state unchanged
        end

        BE->>DB: Sync related post total prices
        BE-->>Buyer: Return HTTP 200 OK
    end
```

---

## 5. Direct Subscription Purchase via PayOS

```mermaid
sequenceDiagram
    autonumber
    actor Member as Member
    participant FE as Frontend Web
    participant BE as Backend Server
    participant PayOS as PayOS Gateway
    participant DB as Database

    Member->>FE: Choose direct Premium purchase
    FE->>BE: POST /api/v1/subscriptions/me/purchase
    BE->>DB: Validate plan, pending payment, downgrade, duplicate unlimited plan

    alt Purchase request is invalid
        BE-->>FE: Return purchase validation error
    else Purchase request is valid
        BE->>DB: Create DepositTransaction with status pending
        BE->>PayOS: CreateCheckoutSession(OrderCode, Amount)
        PayOS-->>BE: Return PaymentUrl
        BE->>DB: Save PaymentUrl to transaction
        BE-->>FE: Return PaymentUrl and OrderCode
        FE-->>Member: Redirect to PayOS payment page
    end

    PayOS->>BE: POST /api/v1/subscriptions/payos-webhook
    BE->>BE: VerifyWebhook signature

    alt Webhook signature is invalid
        BE-->>PayOS: Return webhook error
    else Webhook signature is valid
        BE->>DB: Lock DepositTransaction by OrderCode

        alt Transaction already succeeded
            BE-->>PayOS: Return HTTP 200 OK
        else Transaction is not processed
            BE->>DB: Check paid amount against plan price

            alt Paid amount is invalid
                BE-->>PayOS: Return payment amount error
            else Paid amount is valid
                BE->>DB: Update DepositTransaction to success
                BE->>DB: Lock UserSubscription
                BE->>DB: Apply or extend subscription plan
                BE-->>PayOS: Return HTTP 200 OK
            end
        end
    end
```

---

## 6. Wallet Purchase Flow

```mermaid
sequenceDiagram
    autonumber
    actor Member as Member
    participant FE as Frontend Web
    participant BE as Backend Server
    participant UC as Purchase Use Case
    participant DB as Database

    Member->>FE: Choose Premium purchase with wallet
    FE->>BE: POST /api/v1/subscriptions/me/purchase-with-wallet
    BE->>UC: PurchasePlanWithWallet(UserID, PlanSlug)
    UC->>DB: Lock UserSubscription
    UC->>UC: Validate downgrade and duplicate purchase rules

    alt Purchase is invalid
        UC-->>BE: Return validation error
        BE-->>FE: Return error response
    else Purchase is valid
        UC->>DB: Lock UserWallet
        UC->>DB: Check wallet balance

        alt Wallet balance is insufficient
            UC-->>BE: Return ErrWalletInsufficientBalance
            BE-->>FE: Return HTTP 400
        else Wallet balance is sufficient
            UC->>DB: Deduct plan price from wallet
            UC->>DB: Create wallet statement for subscription purchase
            UC->>DB: Update subscription plan and expiration date
            UC-->>BE: Return purchase success
            BE-->>FE: Return HTTP 200 OK
            FE-->>Member: Show active Premium plan
        end
    end
```

---

## 7. Scheduled Auto-Renewal RenewalWorker

```mermaid
sequenceDiagram
    autonumber
    participant RenewalWorker as Renewal Cron Worker
    participant UC as Renewal Use Case
    participant DB as Database

    RenewalWorker->>UC: ProcessScheduledRenewals()

    loop Batch expired subscriptions by cursor
        UC->>DB: Query active subscriptions with ExpiresAt < Now
        DB-->>UC: Return expired subscriptions

        loop Process each expired subscription
            UC->>DB: Lock UserSubscription
            UC->>UC: Re-check latest subscription state

            alt Auto-renew is disabled
                UC->>DB: Downgrade subscription to Free plan
            else Auto-renew is enabled
                UC->>DB: Lock UserWallet
                UC->>DB: Check wallet balance

                alt Wallet balance is insufficient
                    UC->>DB: Downgrade subscription to Free plan
                else Wallet balance is sufficient
                    UC->>DB: Deduct renewal price from wallet
                    UC->>DB: Create wallet statement for subscription renewal
                    UC->>DB: Extend subscription expiration date
                end
            end
        end
    end
```

---

## 8. Community Hotness and Feed Ranking

```mermaid
sequenceDiagram
    autonumber
    actor Viewer as Member or Guest
    participant FE as Frontend Web
    participant BE as Backend Server
    participant HotnessWorker as Hotness Cron Worker
    participant DB as Database

    HotnessWorker->>DB: Load dirty, recent, and stale high-score posts
    DB-->>HotnessWorker: Return posts to recalculate

    loop Recalculate global hotness
        HotnessWorker->>HotnessWorker: Calculate time-decay GlobalHotnessScore
        HotnessWorker->>DB: Upsert score snapshot and clear dirty flag
    end

    Viewer->>FE: Open hot community feed
    FE->>BE: GET /api/v1/posts?sort=hot

    alt Guest or member without Style Profile
        BE->>DB: Query posts ordered by GlobalHotnessScore
        DB-->>BE: Return SQL-paginated posts
    else Member has Style Profile
        BE->>DB: Get top 1000 hot feed candidates
        DB-->>BE: Return candidate posts
        BE->>DB: Get viewer TasteEmbedding
        DB-->>BE: Return TasteEmbedding

        loop Rank each candidate post
            BE->>BE: Compute StyleScore from item embeddings
            BE->>BE: Blend GlobalScore and StyleScore into FinalScore
        end

        BE->>BE: Sort candidates by FinalScore and slice page
        BE->>DB: Load IsLiked state for visible posts
        DB-->>BE: Return IsLiked state
    end

    BE-->>FE: Return ranked feed response
    FE-->>Viewer: Display community feed
```
